package cf

import (
	"context"

	"github.com/cloudflare/cloudflare-go"
)

func (c *Cloudflare) ZoneID(name string) (string, error) {
	zoneID, err := c.zoneAPI.ZoneIDByName(name)
	if err != nil {
		return zoneID, err
	}
	return zoneID, nil
}

type ZoneFilter func(zone []cloudflare.Zone) []cloudflare.Zone

// GetZones returns a map of zone names to zone IDs. This is useful for
// interacting with the Cloudflare API since many requests (such as DNS
// requests) require a zone ID. Zones can be filtered too based on a given set
// of filters that are ran in the order provided.
func (c *Cloudflare) GetZones(filters ...ZoneFilter) (map[string]string, error) {
	zones, err := c.zoneAPI.ListZones(context.Background())
	if err != nil {
		return nil, err
	}
	for _, filter := range filters {
		zones = filter(zones)
	}
	zoneMap := make(map[string]string)
	for _, zone := range zones {
		zoneMap[zone.Name] = zone.ID
	}
	return zoneMap, nil
}

// ZoneFilterNameIn returns a zone filter that filters zones based on the given
// names. If a zone has a name that is **not** in the given list, it will be
// filtered out of any returned set.
func ZoneFilterNameIn(names ...string) ZoneFilter {
	return func(zones []cloudflare.Zone) []cloudflare.Zone {
		if len(names) == 0 {
			return zones
		}
		filteredZones := make([]cloudflare.Zone, 0)
		for _, zone := range zones {
			for _, name := range names {
				if zone.Name == name {
					filteredZones = append(filteredZones, zone)
				}
			}
		}
		return filteredZones
	}
}

// ZoneFilterIDIn returns a zone filter that filters zones based on the given
// ids. If a zone has an id that is **not** in the given list, it will be
// filtered out of any returned set.
func ZoneFilterIDIn(ids ...string) ZoneFilter {
	return func(zones []cloudflare.Zone) []cloudflare.Zone {
		if len(ids) == 0 {
			return zones
		}
		filteredZones := make([]cloudflare.Zone, 0)
		for _, zone := range zones {
			for _, id := range ids {
				if zone.ID == id {
					filteredZones = append(filteredZones, zone)
				}
			}
		}
		return filteredZones
	}
}
