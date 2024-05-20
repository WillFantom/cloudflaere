package rules

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Traefik struct {
	URL string
}

type TraefikVersion struct {
	Version  string `json:"Version"`
	Codename string `json:"Codename"`
}

func NewTraefik(traefikURL string) (*Traefik, error) {
	tr := &Traefik{URL: traefikURL}
	_, err := tr.GetVersion()
	if err != nil {
		return nil, err
	}
	return tr, nil
}

func (t *Traefik) GetVersion() (*TraefikVersion, error) {
	apiPath, err := url.JoinPath(t.URL, "/api/version")
	if err != nil {
		return nil, fmt.Errorf("could not join url path for the version endpoint: %w", err)
	}
	resp, err := http.Get(apiPath)
	if err != nil {
		return nil, fmt.Errorf("could not fetch version from traefik: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not fetch version from traefik: %s", resp.Status)
	}
	var version TraefikVersion
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return nil, fmt.Errorf("could not decode version response: %w", err)
	}
	return &version, nil
}

func init() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}
