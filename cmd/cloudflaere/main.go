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
	"github.com/willfantom/cloudflaere/pkg/cf"
	"github.com/willfantom/cloudflaere/pkg/tr"
	"github.com/willfantom/cloudflaere/pkg/wtfip"
)

var (
	rootCmd = &cobra.Command{
		Use:   "cloudflaere",
		Short: "cloudflaere is a tool to manage cloudflare dns records based on traefik router rules",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			targetConfig, err := cmd.PersistentFlags().GetString("config")
			if err == nil && targetConfig != "" {
				viper.SetConfigFile(targetConfig)
			}
			if err := viper.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
					return fmt.Errorf("could not read config file: %w", err)
				}
			}
			if viper.GetBool("verbose") {
				logrus.SetLevel(logrus.DebugLevel)
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {

		},
		Run: func(cmd *cobra.Command, args []string) {
			firstRun := true
			for {

				// WAIT IF NOT FIRST RUN
				if firstRun {
					logrus.Infoln("starting cloudflaere loop")
					firstRun = false
				} else {
					logrus.WithField("interval", viper.GetDuration("interval")).Infoln("waiting for next interval")
					<-time.After(viper.GetDuration("interval"))
				}

				// CONFIGURE CLIENTS
				c, err := cf.NewCloudflare(viper.GetString("cloudflare.zone"), viper.GetString("cloudflare.dns"))
				if err != nil {
					logrus.WithError(err).Errorln("cloudflare api client could not be created")
					continue
				}
				t, err := tr.NewTraefik(viper.GetString("traefik.url"))
				if err != nil {
					logrus.WithError(err).Errorln("traefik api client could not be created")
					continue
				}

				// GET DOMAINS FROM TRAEFIK
				trDomains, err := t.GetDomains()
				if err != nil {
					logrus.WithError(err).Errorln("could not fetch domains from traefik")
					continue
				}
				logrus.WithField("count", len(trDomains)).Infoln("domains fetched from traefik")
				if len(trDomains) == 0 {
					logrus.Warnln("no domains found in traefik")
					continue
				}

				// GET ZONES FROM CLOUDFLARE
				cfZones, err := c.GetZones()
				if err != nil {
					logrus.WithError(err).Errorln("could not fetch zones from cloudflare")
					continue
				}
				logrus.WithField("count", len(cfZones)).Infoln("zones fetched from cloudflare")
				if len(cfZones) == 0 {
					logrus.Warnln("no zones found in cloudflare")
					continue
				}

				// DOMAIN ZONE BUCKETS
				domainZones := make(map[string][]string)
				for _, domain := range trDomains {
					logrus.WithField("domain", domain).Debugln("processing domain")
					rootDomain, err := domain.Root()
					if err != nil {
						logrus.WithError(err).WithField("domain", domain).Warnln("could not parse root domain")
						continue
					}
					if _, ok := cfZones[rootDomain]; !ok {
						logrus.WithField("domain", domain).Warnln("root domain is not in cloudflare zones")
						continue
					}
					logrus.WithField("root_domain", rootDomain).Debugln("root domain parsed")
					if _, ok := domainZones[cfZones[rootDomain]]; !ok {
						domainZones[cfZones[rootDomain]] = make([]string, 0)
					}
					domainZones[cfZones[rootDomain]] = append(domainZones[rootDomain], domain.String())
				}

				// CREATE MAGIC COMMENT
				magicCommentKey := viper.GetString("instance")
				if magicCommentKey == "" {
					magicCommentKey, _ = os.Hostname()
				}
				magicComment := fmt.Sprintf("##cloudflaere:%s##", magicCommentKey)

				// GET ADDRESSES
				addresses := make(map[string]netip.Addr)
				if viper.GetBool("ddns.ipv4") {
					ipresp, err := wtfip.LookupIP(false)
					if err != nil {
						logrus.WithError(err).Errorln("could not fetch ipv4 address")
						continue
					}
					ip, err := ipresp.Address()
					if err != nil {
						logrus.WithError(err).WithField("address", ipresp.IPAddress).Errorln("could not parse ipv4 address")
						continue
					}
					addresses["A"] = ip
					logrus.WithField("address", ip.StringExpanded()).Infoln("address v4 fetched")
				}
				if viper.GetBool("ddns.ipv6") {
					ipresp, err := wtfip.LookupIP(true)
					if err != nil {
						logrus.WithError(err).Errorln("could not fetch ipv6 address")
						continue
					}
					ip, err := ipresp.Address()
					if err != nil {
						logrus.WithError(err).WithField("address", ipresp.IPAddress).Errorln("could not parse ipv6 address")
						continue
					}
					addresses["AAAA"] = ip
					logrus.WithField("address", ip.StringExpanded()).Infoln("address v6 fetched")
				}

				// FOR EACH ROOT DOMAIN
				for zoneID, domains := range domainZones {
					records, err := c.GetRecords(zoneID)
					if err != nil {
						logrus.WithError(err).WithField("zone_id", zoneID).WithField("domains", len(domains)).Errorln("could not fetch records from cloudflare")
						continue
					}
					logrus.WithField("zone_id", zoneID).WithField("records", len(records)).Debugln("records fetched from cloudflare")

					// ADD
					for recordType, address := range addresses {
						for _, domain := range domains {
							recs := c.FilterRecords(records, cf.RecordFilterNameIn(domain), cf.RecordFilterTypeIn(recordType))
							if len(recs) == 0 {
								// Record not exist -> create
								if r, err := c.AddRecord(zoneID, recordType, domain, address.StringExpanded(), magicComment, viper.GetBool("proxied")); err != nil {
									logrus.WithError(err).WithField("domain", domain).Errorln("could not add record")
								} else {
									logrus.WithField("record", r).WithField("domain", domain).Debugln("record created")
								}
								continue
							}
							if len(recs) > 1 {
								logrus.WithField("domain", domain).Errorln("more than one record found for domain")
								continue
							}
							if recs[0].Address != address.StringExpanded() && strings.Contains(recs[0].Comment, magicComment) {
								// Record exists but address is different -> update
								if err := c.UpdateRecordAddress(zoneID, recs[0].ID, address.StringExpanded()); err != nil {
									logrus.WithError(err).WithField("domain", domain).Errorln("could not update record")
								} else {
									logrus.WithField("domain", domain).Debugln("record updated")
								}
							} else {
								logrus.WithField("domain", domain).Debugln("record is up to date")
							}
						}
					}

					// CLEAN
					for _, record := range records {
						if strings.Contains(record.Comment, magicComment) {
							hasDomain := false
							for _, domain := range domains {
								if strings.EqualFold(record.Name, domain) {
									hasDomain = true
									break
								}
							}
							if !hasDomain {
								// Record exists but domain is not in traefik -> delete
								if err := c.DeleteRecord(zoneID, record.ID); err != nil {
									logrus.WithError(err).WithField("record", record).Errorln("could not delete record")
								} else {
									logrus.WithField("record", record).Debugln("record deleted")
								}
							}
						}
					}
				}
			}
		},
	}
)

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

	// config path override
	rootCmd.PersistentFlags().String("config", "", "config file path")
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	// globals
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "output debug logs")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	rootCmd.PersistentFlags().Duration("interval", time.Minute, "interval between checks")
	viper.BindPFlag("interval", rootCmd.PersistentFlags().Lookup("interval"))
	rootCmd.PersistentFlags().String("instance", "", "unique name of cloudflaere instance (default $HOSTNAME)")
	viper.BindPFlag("instance", rootCmd.PersistentFlags().Lookup("instance"))

	// traefik
	rootCmd.PersistentFlags().String("tr-url", "", "target traefik url (e.g. https://traefik.example.com)")
	viper.BindPFlag("traefik.url", rootCmd.PersistentFlags().Lookup("tr-url"))

	// cloudflare
	rootCmd.PersistentFlags().Bool("cf-proxied", false, "set new records to be proxied by cloudflare")
	viper.BindPFlag("cloudflare.proxied", rootCmd.PersistentFlags().Lookup("proxied"))
	rootCmd.PersistentFlags().String("cf-zone", "", "cloudflare zone read api key")
	viper.BindPFlag("cloudflare.zone", rootCmd.PersistentFlags().Lookup("cf-zone"))
	rootCmd.PersistentFlags().String("cf-dns", "", "cloudflare dns edit api key")
	viper.BindPFlag("cloudflare.dns", rootCmd.PersistentFlags().Lookup("cf-dns"))

	// ddns
	rootCmd.PersistentFlags().BoolP("ipv4", "4", false, "enable ipv4 ddns")
	viper.BindPFlag("ddns.ipv4", rootCmd.PersistentFlags().Lookup("ipv4"))
	rootCmd.PersistentFlags().BoolP("ipv6", "6", false, "enable ipv6 ddns")
	viper.BindPFlag("ddns.ipv6", rootCmd.PersistentFlags().Lookup("ipv6"))

	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	viper.AutomaticEnv()
}
