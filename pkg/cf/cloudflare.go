package cf

import (
	"context"
	"sync"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

type Cloudflare struct {
	lock    *sync.RWMutex
	zoneAPI *cloudflare.API
	dnsAPI  *cloudflare.API

	allowedZones ZoneFilter

	records map[string]*Record
}

// NewCloudflare creates a new Cloudflare API client for both the zone and DNS
// API. This returns instances to the given APIs and any errors.
// TODO: check api keys
func NewCloudflare(zoneKey, dnsKey string) (*Cloudflare, error) {
	cfZone, err := cloudflare.NewWithAPIToken(zoneKey)
	if err != nil {
		return nil, err
	}
	cfDNS, err := cloudflare.NewWithAPIToken(dnsKey)
	if err != nil {
		return nil, err
	}
	return &Cloudflare{
		lock:    &sync.RWMutex{},
		zoneAPI: cfZone,
		dnsAPI:  cfDNS,
		records: make(map[string]*Record),
	}, nil
}

func (c *Cloudflare) SetAllowedZones(names ...string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.allowedZones = ZoneFilterNameIn(names...)
}

func (c *Cloudflare) Records() map[string]*Record {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.records
}

func (c *Cloudflare) Poll(ctx context.Context, interval time.Duration) <-chan error {
	errChan := make(chan error)
	go func() {
		for {
			c.lock.Lock()
			zones, err := c.GetZones(c.allowedZones)
			if err != nil {
				errChan <- err
			}
			c.records = make(map[string]*Record)
			for _, zone := range zones {
				records, err := c.GetRecords(zone, RecordFilterTypeIn("A", "AAAA"))
				if err != nil {
					errChan <- err
				}
				c.records = make(map[string]*Record)
				for _, record := range records {
					c.records[zone] = record
				}
			}
			c.lock.Unlock()
			select {
			case <-ctx.Done():
				errChan <- nil
				return
			case <-time.After(interval):
				continue
			}
		}
	}()
	return errChan
}
