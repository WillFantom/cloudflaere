package cf

import (
	"context"
	"net/netip"
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

type Record struct {
	ID      string
	Type    string
	Name    string
	Address string
	Comment string
	Proxied bool
}

func newRecord(id, t, name, address, comment string, proxied bool) *Record {
	return &Record{
		ID:      id,
		Type:    t,
		Name:    name,
		Address: address,
		Comment: comment,
		Proxied: proxied,
	}
}

type RecordFilter func(record []*Record) []*Record

func (c *Cloudflare) GetRecords(zoneID string, filters ...RecordFilter) ([]*Record, error) {
	records, _, err := c.dnsAPI.ListDNSRecords(
		context.Background(),
		cloudflare.ZoneIdentifier(zoneID),
		cloudflare.ListDNSRecordsParams{}, // TODO: Apply filters here perhaps?
	)
	if err != nil {
		return nil, err
	}
	filteredRecords := make([]*Record, len(records))
	for i, record := range records {
		filteredRecords[i] = newRecord(record.ID, record.Type, record.Name, record.Content, record.Comment, *record.Proxied)
	}
	return c.FilterRecords(filteredRecords, filters...), nil
}

func (c *Cloudflare) FilterRecords(records []*Record, filters ...RecordFilter) []*Record {
	filteredRecords := records
	for _, filter := range filters {
		filteredRecords = filter(filteredRecords)
	}
	return filteredRecords
}

func (c *Cloudflare) AddRecord(zoneID string, t, name, content, comment string, proxied bool) (*Record, error) {
	if r, err := c.dnsAPI.CreateDNSRecord(
		context.Background(),
		cloudflare.ZoneIdentifier(zoneID),
		cloudflare.CreateDNSRecordParams{
			Type:    t,
			Name:    name,
			Content: content,
			Proxied: cloudflare.BoolPtr(proxied),
			Comment: comment,
		},
	); err != nil {
		return nil, err
	} else {
		return newRecord(r.ID, r.Type, r.Name, r.Content, r.Comment, *r.Proxied), nil
	}
}

func (c *Cloudflare) UpdateRecordAddress(zoneID string, id, address string) error {
	_, err := c.dnsAPI.UpdateDNSRecord(
		context.Background(),
		cloudflare.ZoneIdentifier(zoneID),
		cloudflare.UpdateDNSRecordParams{
			ID:      id,
			Content: address,
		},
	)
	return err
}

func (c *Cloudflare) DeleteRecord(zoneID string, id string) error {
	return c.dnsAPI.DeleteDNSRecord(
		context.Background(),
		cloudflare.ZoneIdentifier(zoneID),
		id,
	)
}

func RecordFilterTypeIn(types ...string) RecordFilter {
	return func(records []*Record) []*Record {
		if len(types) == 0 {
			return records
		}
		filteredRecords := make([]*Record, 0)
		for _, record := range records {
			for _, t := range types {
				if record.Type == t {
					filteredRecords = append(filteredRecords, record)
				}
			}
		}
		return filteredRecords
	}
}

func RecordFilterNameIn(names ...string) RecordFilter {
	return func(records []*Record) []*Record {
		if len(names) == 0 {
			return records
		}
		filteredRecords := make([]*Record, 0)
		for _, record := range records {
			for _, n := range names {
				if record.Name == n {
					filteredRecords = append(filteredRecords, record)
				}
			}
		}
		return filteredRecords
	}
}

func RecordFilterAddressIn(addresses ...netip.Addr) RecordFilter {
	return func(records []*Record) []*Record {
		if len(addresses) == 0 {
			return records
		}
		filteredRecords := make([]*Record, 0)
		for _, record := range records {
			rAddress, err := netip.ParseAddr(record.Address)
			if err != nil {
				continue
			}
			for _, address := range addresses {
				if rAddress.Compare(address) == 0 {
					filteredRecords = append(filteredRecords, record)
				}
			}
		}
		return filteredRecords
	}
}

func RecordFilterCommentContains(str string) RecordFilter {
	return func(records []*Record) []*Record {
		if len(str) == 0 {
			return records
		}
		filteredRecords := make([]*Record, 0)
		for _, record := range records {
			if record.Comment != "" && strings.Contains(record.Comment, str) {
				filteredRecords = append(filteredRecords, record)
			}
		}
		return filteredRecords
	}
}

// import (
// 	"context"
// 	"fmt"
// 	"net/netip"
// 	"strings"

// 	"github.com/cloudflare/cloudflare-go"
// )

// type Record struct {
// 	ID      string
// 	Name    string
// 	Content string
// 	Proxied bool
// 	Type    string
// 	Comment string
// }

// func NewRecord(domain, address, comment string, proxied bool) (*Record, error) {
// 	ipAddr, err := netip.ParseAddr(address)
// 	if err != nil {
// 		return nil, fmt.Errorf("could not parse address: %w", err)
// 	}
// 	recordType := "A"
// 	if ipAddr.Is6() {
// 		recordType = "AAAA"
// 	} else if ipAddr.Is4() {
// 		recordType = "A"
// 	} else {
// 		return nil, fmt.Errorf("unsupported address type: %s", address)
// 	}
// 	return &Record{
// 		Name:    domain,
// 		Content: ipAddr.StringExpanded(),
// 		Proxied: proxied,
// 		Comment: comment,
// 		Type:    recordType,
// 	}, nil
// }

// func (r Record) CommentContains(comment string) bool {
// 	return strings.Contains(r.Comment, comment)
// }

// // NewRecord adds a new DNS record to the zone.
// func (z *Zone) NewRecord(domain, address, comment string, proxied bool) (*Record, error) {
// 	r, err := NewRecord(domain, address, comment, proxied)
// 	if err != nil {
// 		return nil, err
// 	}
// 	record, err := z.cfAPI.CreateDNSRecord(
// 		context.Background(),
// 		cloudflare.ZoneIdentifier(z.zoneID),
// 		cloudflare.CreateDNSRecordParams{
// 			Type:    r.Type,
// 			Name:    r.Name,
// 			Content: r.Content,
// 			Proxied: cloudflare.BoolPtr(r.Proxied),
// 			Comment: r.Comment,
// 		},
// 	)
// 	return &Record{
// 		ID:      record.ID,
// 		Name:    record.Name,
// 		Content: record.Content,
// 		Proxied: *record.Proxied,
// 		Comment: record.Comment,
// 	}, err
// }

// func (z *Zone) UpdateRecord(id, domain, address, comment string) (*Record, error) {
// 	r, err := NewRecord(domain, address, comment, false)
// 	if err != nil {
// 		return nil, err
// 	}
// 	record, err := z.cfAPI.UpdateDNSRecord(
// 		context.Background(),
// 		cloudflare.ZoneIdentifier(z.zoneID),
// 		cloudflare.UpdateDNSRecordParams{
// 			ID:      id,
// 			Type:    r.Type,
// 			Name:    r.Name,
// 			Content: r.Content,
// 			Comment: cloudflare.StringPtr(r.Comment),
// 		},
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("could not update record: %w", err)
// 	}
// 	return &Record{
// 		ID:      record.ID,
// 		Name:    record.Name,
// 		Content: record.Content,
// 		Proxied: *record.Proxied,
// 		Comment: record.Comment,
// 	}, err
// }

// func (z *Zone) DeleteRecord(id string) error {
// 	err := z.cfAPI.DeleteDNSRecord(
// 		context.Background(),
// 		cloudflare.ZoneIdentifier(z.zoneID),
// 		id,
// 	)
// 	if err != nil {
// 		return fmt.Errorf("could not delete record: %w", err)
// 	}
// 	return nil
// }
