package main

import (
	"fmt"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/willfantom/cloudflaere/pkg/dns"
	"github.com/willfantom/cloudflaere/pkg/rules"
	"golang.org/x/net/publicsuffix"
)

var (
	rootCmd = &cobra.Command{
		Use:   "cloudflaere",
		Short: "Cloudflaere is a tool to manage Cloudflare DNS records based on Traefik router rules",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			targetConfig, err := cmd.PersistentFlags().GetString("config")
			if err == nil && targetConfig != "" {
				viper.SetConfigFile(targetConfig)
			}
			if err := viper.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok {
					return nil
				}
				return fmt.Errorf("could not read config file: %w", err)
			}
			if viper.GetBool("verbose") {
				logrus.SetLevel(logrus.DebugLevel)
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			if viper.GetString("email") == "" {
				logrus.Fatalln("email is required")
			}
			if viper.GetString("apikey") == "" {
				logrus.Fatalln("apikey is required")
			}
			if viper.GetString("zoneid") == "" {
				logrus.Fatalln("zoneid is required")
			}
			if viper.GetString("traefikurl") == "" {
				logrus.Fatalln("traefikurl is required")
			}
			if viper.GetString("address") == "" {
				logrus.Fatalln("address is required")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			// establish cloudflare api connaction
			z, err := dns.NewZone(viper.GetString("zoneid"), viper.GetString("apikey"), viper.GetString("email"))
			if err != nil {
				logrus.WithError(err).Fatalln("cloudflare zone api client could not be created")
			}
			cfZoneName, err := z.GetName()
			if err != nil {
				logrus.WithError(err).Fatalln("could not fetch zone name from cloudflare")
			}
			logrus.WithField("zone_name", cfZoneName).Infoln("cloudflare zone api client created")

			// establish traefik api connection
			tr, err := rules.NewTraefik(viper.GetString("traefikurl"))
			if err != nil {
				logrus.WithError(err).Fatalln("traefik api client could not be created")
			}
			trVersion, err := tr.GetVersion()
			if err != nil {
				logrus.WithError(err).Fatalln("could not fetch traefik version")
			}
			logrus.WithField("version", trVersion.Codename).Infoln("traefik api client created")

			// get ip mode
			address, err := netip.ParseAddr(viper.GetString("address"))
			if err != nil {
				logrus.WithError(err).Fatalln("could not parse address")
			}
			recordType := "A"
			if address.Is6() {
				recordType = "AAAA"
			} else if address.Is4() {
				recordType = "A"
			} else {
				logrus.WithField("address", address.StringExpanded()).Fatalln("unsupported address type")
			}
			logrus.WithField("address", address.StringExpanded()).WithField("type", recordType).Infoln("address parsed")

			// set magic comment
			hostname, err := os.Hostname()
			if err != nil {
				logrus.WithError(err).Fatalln("could not fetch hostname")
			}
			magicComment := fmt.Sprintf("##cloudflaere:%s##", hostname)
			logrus.WithField("magic_comment", magicComment).Infoln("using magic comment")

			firstRun := true

			for {
				if firstRun {
					logrus.Infoln("starting cloudflaere loop")
					firstRun = false
				} else {
					<-time.After(viper.GetDuration("interval"))
				}

				// get all domains from traefik
				domains, err := tr.GetDomains()
				if err != nil {
					logrus.WithError(err).Errorln("could not fetch domains from traefik")
					continue
				}
				if len(domains) == 0 {
					logrus.Warnln("no domains found in traefik")
					continue
				} else {
					logrus.WithField("domains", domains).Debugln("domains fetched from traefik")
				}

				// filter to only supported domains for dns mgmt
				supportedDomains := make([]string, 0)
				for _, domain := range domains {
					rootDomain, err := publicsuffix.EffectiveTLDPlusOne(domain)
					if err != nil {
						logrus.WithError(err).WithField("domain", domain).Warnln("could not parse root domain")
						continue
					}
					if strings.EqualFold(rootDomain, cfZoneName) {
						supportedDomains = append(supportedDomains, domain)
					}
				}
				if len(supportedDomains) == 0 {
					logrus.Warnln("no supportedDomains were found in traefik")
					continue
				} else {
					logrus.WithField("domains", supportedDomains).WithField("root_domain", cfZoneName).Debugln("supported domains fetched from traefik")
				}

				// get all dns records from cloudflare
				records, err := z.GetRecords(dns.RecordsWithType(recordType))
				if err != nil {
					logrus.WithError(err).Errorln("could not fetch dns records from cloudflare")
					continue
				}
				if len(records) == 0 {
					logrus.WithField("type", recordType).Warnln("no records found in cloudflare")
					continue
				} else {
					logrus.WithField("type", recordType).WithField("records", records).Debugln("records fetched from cloudflare")
				}

				// add/update records based on traefik routers
				for _, domain := range supportedDomains {
					records := getDomainRecords(domain, records)
					if len(records) > 1 {
						logrus.WithField("domain", domain).Errorln("more than one record found for domain")
						continue
					}

					if len(records) == 0 {
						// create record
						if r, err := z.NewRecord(domain, address.StringExpanded(), magicComment, viper.GetBool("proxied")); err != nil {
							logrus.WithError(err).WithField("domain", domain).Errorln("could not add record")
						} else {
							logrus.WithField("record", r).WithField("domain", domain).Debugln("record created")
						}
						continue
					} else if len(records) == 1 {
						if !records[0].CommentContains(magicComment) {
							logrus.WithField("domain", domain).Warnln("pre-existing record does not contain magic comment")
							continue
						} else {
							if records[0].Content != address.StringExpanded() {
								// update record
								r, err := z.UpdateRecord(records[0].ID, domain, address.StringExpanded(), magicComment)
								if err != nil {
									logrus.WithError(err).WithField("domain", domain).Errorln("could not update record")
								} else {
									logrus.WithField("record", r).WithField("domain", domain).Debugln("record updated")
								}
							} else {
								logrus.WithField("domain", domain).Debugln("record is up to date")
							}
						}
					}
				}
				logrus.Infoln("records updated")

				// remove records that do not have a traefik router
				for _, record := range records {
					if record.CommentContains(magicComment) {
						hasDomain := false
						for _, domain := range supportedDomains {
							if strings.EqualFold(record.Name, domain) {
								hasDomain = true
								break
							}
						}
						if !hasDomain {
							// delete record
							if err := z.DeleteRecord(record.ID); err != nil {
								logrus.WithError(err).WithField("record", record).Errorln("could not delete record")
							} else {
								logrus.WithField("record", record).Debugln("record deleted")
							}
						}
					}
				}
				logrus.Infoln("records cleaned")
			}
		},
	}
)

func getDomainRecords(domain string, records []*dns.Record) []*dns.Record {
	domainRecords := make([]*dns.Record, 0)
	for _, record := range records {
		if record.Name == domain {
			domainRecords = append(domainRecords, record)
		}
	}
	return domainRecords
}
func main() {

	rootCmd.Execute()

}

func init() {
	viper.SetConfigName("cloudflaere")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.cloudflaere")
	viper.AddConfigPath("/etc/cloudflaere")
	viper.AddConfigPath("$HOME/.config/cloudflaere")

	rootCmd.PersistentFlags().String("config", "", "config file path")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "output debug logs")
	rootCmd.PersistentFlags().String("email", "", "cloudflare email address")
	rootCmd.PersistentFlags().String("apikey", "", "cloudflare global api key")
	rootCmd.PersistentFlags().String("zoneid", "", "cloudflare zone id")
	rootCmd.PersistentFlags().String("traefikurl", "https://traefik.example.com", "target traefik url")
	rootCmd.PersistentFlags().String("address", "", "ipv4 or ipv6 address to set as the target")
	rootCmd.PersistentFlags().Bool("proxied", false, "use cf proxy by default on new dns record")
	rootCmd.PersistentFlags().DurationP("interval", "i", 30*time.Second, "interval to check for updates")
	viper.BindPFlags(rootCmd.PersistentFlags())

	viper.EnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
