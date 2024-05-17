package dns

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

type Record struct {
	ID      string
	Name    string
	Content string
	Proxied bool
	Type    string
	Comment string
}

func NewRecord(domain, address, comment string, proxied bool) (*Record, error) {
	ipAddr, err := netip.ParseAddr(address)
	if err != nil {
		return nil, fmt.Errorf("could not parse address: %w", err)
	}
	recordType := "A"
	if ipAddr.Is6() {
		recordType = "AAAA"
	} else if ipAddr.Is4() {
		recordType = "A"
	} else {
		return nil, fmt.Errorf("unsupported address type: %s", address)
	}
	return &Record{
		Name:    domain,
		Content: ipAddr.StringExpanded(),
		Proxied: proxied,
		Comment: comment,
		Type:    recordType,
	}, nil
}

func (r Record) CommentContains(comment string) bool {
	return strings.Contains(r.Comment, comment)
}

// NewRecord adds a new DNS record to the zone.
func (z *Zone) NewRecord(domain, address, comment string, proxied bool) (*Record, error) {
	r, err := NewRecord(domain, address, comment, proxied)
	if err != nil {
		return nil, err
	}
	record, err := z.cfAPI.CreateDNSRecord(
		context.Background(),
		cloudflare.ZoneIdentifier(z.zoneID),
		cloudflare.CreateDNSRecordParams{
			Type:    r.Type,
			Name:    r.Name,
			Content: r.Content,
			Proxied: cloudflare.BoolPtr(r.Proxied),
			Comment: r.Comment,
		},
	)
	return &Record{
		ID:      record.ID,
		Name:    record.Name,
		Content: record.Content,
		Proxied: *record.Proxied,
		Comment: record.Comment,
	}, err
}

func (z *Zone) UpdateRecord(id, domain, address, comment string) (*Record, error) {
	r, err := NewRecord(domain, address, comment, false)
	if err != nil {
		return nil, err
	}
	record, err := z.cfAPI.UpdateDNSRecord(
		context.Background(),
		cloudflare.ZoneIdentifier(z.zoneID),
		cloudflare.UpdateDNSRecordParams{
			ID:      id,
			Type:    r.Type,
			Name:    r.Name,
			Content: r.Content,
			Comment: cloudflare.StringPtr(r.Comment),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not update record: %w", err)
	}
	return &Record{
		ID:      record.ID,
		Name:    record.Name,
		Content: record.Content,
		Proxied: *record.Proxied,
		Comment: record.Comment,
	}, err
}

func (z *Zone) DeleteRecord(id string) error {
	err := z.cfAPI.DeleteDNSRecord(
		context.Background(),
		cloudflare.ZoneIdentifier(z.zoneID),
		id,
	)
	if err != nil {
		return fmt.Errorf("could not delete record: %w", err)
	}
	return nil
}
