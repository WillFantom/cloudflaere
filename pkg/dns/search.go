package dns

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

type Filter func(record []cloudflare.DNSRecord) []cloudflare.DNSRecord

func (z *Zone) GetRecords(filters ...Filter) ([]*Record, error) {
	cfARecords, _, err := z.cfAPI.ListDNSRecords(
		context.Background(),
		cloudflare.ZoneIdentifier(z.zoneID),
		cloudflare.ListDNSRecordsParams{
			Type: "A",
		},
	)
	cfAAAARecords, _, err := z.cfAPI.ListDNSRecords(
		context.Background(),
		cloudflare.ZoneIdentifier(z.zoneID),
		cloudflare.ListDNSRecordsParams{
			Type: "AAAA",
		},
	)
	cfRecords := append(cfARecords, cfAAAARecords...)
	if err != nil {
		return nil, fmt.Errorf("could not fetch dns records from cloudflare: %w", err)
	}
	for _, filter := range filters {
		cfRecords = filter(cfRecords)
	}
	records := make([]*Record, len(cfRecords))
	for i, record := range cfRecords {
		r, err := NewRecord(record.Name, record.Content, record.Comment, *record.Proxied)
		if err != nil {
			return nil, fmt.Errorf("could not create record: %w", err)
		}
		records[i] = r
		records[i].ID = record.ID
	}
	return records, nil
}

func RecordsWithComment(comment string) Filter {
	return func(records []cloudflare.DNSRecord) []cloudflare.DNSRecord {
		filteredRecords := make([]cloudflare.DNSRecord, 0)
		for _, record := range records {
			if strings.Contains(record.Comment, comment) {
				filteredRecords = append(filteredRecords, record)
			}
		}
		return filteredRecords
	}
}

func RecordsWithName(name string) Filter {
	return func(records []cloudflare.DNSRecord) []cloudflare.DNSRecord {
		filteredRecords := make([]cloudflare.DNSRecord, 0)
		for _, record := range records {
			if strings.EqualFold(record.Name, name) {
				filteredRecords = append(filteredRecords, record)
			}
		}
		return filteredRecords
	}
}

func RecordsWithType(recordType string) Filter {
	return func(records []cloudflare.DNSRecord) []cloudflare.DNSRecord {
		filteredRecords := make([]cloudflare.DNSRecord, 0)
		for _, record := range records {
			if strings.EqualFold(record.Type, recordType) {
				filteredRecords = append(filteredRecords, record)
			}
		}
		return filteredRecords
	}
}
