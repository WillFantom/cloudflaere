package wtfip

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
)

type LookupResponse struct {
	IPAddress   string `json:"YourFuckingIPAddress"`
	Location    string `json:"YourFuckingLocation"`
	Hostname    string `json:"YourFuckingHostname"`
	ISP         string `json:"YourFuckingISP"`
	TorExit     bool   `json:"YourFuckingTorExit"`
	City        string `json:"YourFuckingCity"`
	Country     string `json:"YourFuckingCountry"`
	CountryCode string `json:"YourFuckingCountryCode"`
}

func LookupIP(ipv6 bool) (*LookupResponse, error) {
	lookupURL := "https://ipv4.wtfismyip.com/json"
	if ipv6 {
		lookupURL = "https://ipv6.wtfismyip.com/json"
	}
	resp, err := http.Get(lookupURL)
	if err != nil {
		return nil, fmt.Errorf("")
	}
	defer resp.Body.Close()
	var lookupResp LookupResponse
	err = json.NewDecoder(resp.Body).Decode(&lookupResp)
	if err != nil {
		return nil, err
	}
	return &lookupResp, nil
}

func (lr LookupResponse) Address() (netip.Addr, error) {
	addr, err := netip.ParseAddr(lr.IPAddress)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("failed to parse found address: %w", err)
	}
	return addr, nil
}
