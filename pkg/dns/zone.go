package dns

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// Zone represents the means to interact with a Cloudflare zone. This is a
// combination of a cloudfalre API client and a zone ID. Functions on this type
// will always use the zone ID to interact with the Cloudflare API.
type Zone struct {
	cfAPI  *cloudflare.API
	zoneID string
}

func NewZone(zoneID, key, email string) (*Zone, error) {
	cfAPI, err := cloudflare.New(key, email)
	if err != nil {
		return nil, err
	}
	z := &Zone{
		cfAPI:  cfAPI,
		zoneID: zoneID,
	}
	if _, err := z.GetName(); err != nil {
		return nil, fmt.Errorf("could not fetch zone name from cloudflare: %w", err)
	}
	return z, nil
}

// GetName returns the name of the zone based on the zone ID.
func (z *Zone) GetName() (string, error) {
	cfZone, err := z.cfAPI.ZoneDetails(context.Background(), z.zoneID)
	if err != nil {
		return "", fmt.Errorf("zone information could not be queried: %w", err)
	}
	return cfZone.Name, nil
}
