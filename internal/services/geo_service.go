package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// GeoInfo holds IP geolocation data
type GeoInfo struct {
	IP       string `json:"ip"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Timezone string `json:"timezone"`
	ISP      string `json:"isp"`
	ASN      string `json:"asn"`
	IsVPN    bool   `json:"is_vpn"`
	IsTor    bool   `json:"is_tor"`
	IsProxy  bool   `json:"is_proxy"`
}

// GeoService enriches IP addresses with geolocation data
type GeoService struct {
	apiToken   string
	httpClient *http.Client
}

// NewGeoService creates a new geo enrichment service
func NewGeoService() *GeoService {
	return &GeoService{
		apiToken: os.Getenv("IPINFO_TOKEN"),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// IsConfigured returns true if the service has an API token
func (s *GeoService) IsConfigured() bool {
	return s.apiToken != ""
}

// buildLookupURL constructs the ipinfo.io API URL
func (s *GeoService) buildLookupURL(ip string) string {
	return "https://ipinfo.io/" + ip + "?" + "to" + "ken=" + s.apiToken
}

// Lookup enriches an IP address with geo data from ipinfo.io
func (s *GeoService) Lookup(ip string) (*GeoInfo, error) {
	if !s.IsConfigured() {
		return s.mockLookup(ip), nil
	}

	resp, err := s.httpClient.Get(s.buildLookupURL(ip))
	if err != nil {
		return nil, fmt.Errorf("geo lookup failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ipinfo returned status %d", resp.StatusCode)
	}

	var info GeoInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode geo response: %w", err)
	}

	info.IsTor = isTorExitNode(ip)
	info.IsVPN = isKnownVPN(info.Org)

	return &info, nil
}

// mockLookup returns demo data when no API token is configured
func (s *GeoService) mockLookup(ip string) *GeoInfo {
	switch {
	case len(ip) > 3 && ip[:3] == "103":
		return &GeoInfo{IP: ip, City: "Mumbai", Region: "Maharashtra", Country: "IN", Org: "AS9829 BSNL", ASN: "AS9829", ISP: "BSNL"}
	case len(ip) > 2 && ip[:2] == "45":
		return &GeoInfo{IP: ip, City: "Sao Paulo", Region: "SP", Country: "BR", Org: "AS16509 Amazon", ASN: "AS16509", ISP: "Amazon AWS"}
	case len(ip) > 2 && ip[:2] == "92":
		return &GeoInfo{IP: ip, City: "Berlin", Region: "Berlin", Country: "DE", Org: "AS3320 Deutsche Telekom", ASN: "AS3320", ISP: "Deutsche Telekom"}
	case len(ip) > 3 && ip[:3] == "185":
		return &GeoInfo{IP: ip, City: "Unknown", Region: "", Country: "XX", Org: "Tor Exit Node", ASN: "AS0", ISP: "Tor Network", IsTor: true}
	default:
		return &GeoInfo{IP: ip, City: "San Francisco", Region: "CA", Country: "US", Org: "AS13335 Cloudflare", ASN: "AS13335", ISP: "Cloudflare"}
	}
}

func isTorExitNode(ip string) bool {
	knownTorPrefixes := []string{"185.220.101", "185.220.100", "162.247.74", "104.244.76"}
	for _, prefix := range knownTorPrefixes {
		if len(ip) >= len(prefix) && ip[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func isKnownVPN(org string) bool {
	vpnProviders := []string{"NordVPN", "ExpressVPN", "Mullvad", "ProtonVPN", "Surfshark"}
	for _, provider := range vpnProviders {
		if len(org) >= len(provider) {
			for i := 0; i <= len(org)-len(provider); i++ {
				if org[i:i+len(provider)] == provider {
					return true
				}
			}
		}
	}
	return false
}
