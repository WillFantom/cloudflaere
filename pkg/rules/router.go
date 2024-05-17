package rules

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	muxer "github.com/traefik/traefik/v3/pkg/muxer/http"
)

type TraefikRouter struct {
	Name    string `json:"service"`
	RuleStr string `json:"rule"`
	Status  string `json:"status"`
}

func (t *Traefik) GetDomains() ([]string, error) {
	apiPath, err := url.JoinPath(t.URL, "/api/http/routers")
	if err != nil {
		return nil, fmt.Errorf("could not join url path for the http routers endpoint: %w", err)
	}
	resp, err := http.Get(apiPath)
	if err != nil {
		return nil, fmt.Errorf("could not fetch routers from traefik: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not fetch routers from traefik: %s", resp.Status)
	}
	var routers []TraefikRouter
	if err := json.NewDecoder(resp.Body).Decode(&routers); err != nil {
		return nil, fmt.Errorf("could not decode routers response: %w", err)
	}
	domains := make([]string, 0)
	for _, router := range routers {
		ds, err := muxer.ParseDomains(router.RuleStr)
		if err != nil {
			return nil, fmt.Errorf("could not parse domains from rule: %w", err)
		}
		domains = append(domains, ds...)
	}
	return domains, nil
}
