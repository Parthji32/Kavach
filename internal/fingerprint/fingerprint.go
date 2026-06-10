package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// CapturedFingerprint contains all data captured when a token is triggered
type CapturedFingerprint struct {
	// Network info
	IPAddress  string `json:"ip_address"`
	Port       string `json:"port"`
	
	// Geo info (enriched via ipinfo.io)
	Country    string `json:"country"`
	City       string `json:"city"`
	Region     string `json:"region"`
	ISP        string `json:"isp"`
	ASN        string `json:"asn"`
	IsVPN      bool   `json:"is_vpn"`
	IsTor      bool   `json:"is_tor"`
	IsProxy    bool   `json:"is_proxy"`

	// Browser/Device info
	UserAgent  string `json:"user_agent"`
	Browser    string `json:"browser"`
	BrowserVer string `json:"browser_version"`
	OS         string `json:"os"`
	OSVersion  string `json:"os_version"`
	Device     string `json:"device"`

	// HTTP fingerprint
	AcceptLang    string            `json:"accept_language"`
	AcceptEnc     string            `json:"accept_encoding"`
	Referrer      string            `json:"referrer"`
	Headers       map[string]string `json:"headers"`

	// TLS fingerprint (JA3/JA4)
	TLSFingerprint string `json:"tls_fingerprint"`
	TLSVersion     string `json:"tls_version"`

	// Client-side fingerprint (from JS if available)
	CanvasHash     string `json:"canvas_hash"`
	WebGLHash      string `json:"webgl_hash"`
	ScreenRes      string `json:"screen_resolution"`
	Timezone       string `json:"timezone"`
	Languages      string `json:"languages"`

	// Computed
	UniqueHash     string `json:"unique_hash"` // Combined fingerprint hash
}

// CaptureFromRequest extracts all available fingerprint data from an HTTP request
func CaptureFromRequest(r *http.Request) *CapturedFingerprint {
	fp := &CapturedFingerprint{
		Headers: make(map[string]string),
	}

	// Extract IP (handle X-Forwarded-For, CF-Connecting-IP, etc.)
	fp.IPAddress = extractRealIP(r)
	
	// Extract port
	_, port, _ := net.SplitHostPort(r.RemoteAddr)
	fp.Port = port

	// HTTP headers
	fp.UserAgent = r.Header.Get("User-Agent")
	fp.AcceptLang = r.Header.Get("Accept-Language")
	fp.AcceptEnc = r.Header.Get("Accept-Encoding")
	fp.Referrer = r.Header.Get("Referer")

	// Capture all headers (for advanced fingerprinting)
	for key, values := range r.Header {
		if len(values) > 0 {
			fp.Headers[key] = values[0]
		}
	}

	// Parse User-Agent for browser/OS
	fp.Browser, fp.BrowserVer = ParseUserAgentBrowser(fp.UserAgent)
	fp.OS, fp.OSVersion = ParseUserAgentOS(fp.UserAgent)

	// Generate unique fingerprint hash
	fp.UniqueHash = fp.GenerateHash()

	return fp
}

// extractRealIP gets the real client IP, considering proxies and CDNs
func extractRealIP(r *http.Request) string {
	// Priority order for IP extraction:
	// 1. CF-Connecting-IP (Cloudflare)
	// 2. X-Real-IP (nginx)
	// 3. X-Forwarded-For (first IP in chain)
	// 4. RemoteAddr (direct connection)

	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Take the first IP (original client)
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	
	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// GenerateHash creates a unique fingerprint hash combining multiple signals
func (fp *CapturedFingerprint) GenerateHash() string {
	// Combine stable signals that identify a device
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		fp.UserAgent,
		fp.AcceptLang,
		fp.AcceptEnc,
		fp.CanvasHash,
		fp.WebGLHash,
		fp.ScreenRes,
	)
	
	hash := sha256.Sum256([]byte(data))
	return "fp_" + hex.EncodeToString(hash[:12]) // 24-char hex hash with prefix
}

// ToJSON serializes the fingerprint to JSON
func (fp *CapturedFingerprint) ToJSON() (string, error) {
	data, err := json.Marshal(fp)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ParseUserAgentBrowser extracts browser name and version from User-Agent
func ParseUserAgentBrowser(ua string) (string, string) {
	ua = strings.ToLower(ua)
	
	if strings.Contains(ua, "chrome") && !strings.Contains(ua, "edg") {
		return "Chrome", extractVersion(ua, "chrome/")
	}
	if strings.Contains(ua, "firefox") {
		return "Firefox", extractVersion(ua, "firefox/")
	}
	if strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome") {
		return "Safari", extractVersion(ua, "version/")
	}
	if strings.Contains(ua, "edg") {
		return "Edge", extractVersion(ua, "edg/")
	}
	
	return "Unknown", ""
}

// ParseUserAgentOS extracts OS name and version from User-Agent
func ParseUserAgentOS(ua string) (string, string) {
	ua = strings.ToLower(ua)
	
	if strings.Contains(ua, "windows") {
		return "Windows", ""
	}
	if strings.Contains(ua, "mac os") || strings.Contains(ua, "macos") {
		return "macOS", ""
	}
	if strings.Contains(ua, "linux") {
		return "Linux", ""
	}
	if strings.Contains(ua, "android") {
		return "Android", ""
	}
	if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
		return "iOS", ""
	}
	
	return "Unknown", ""
}

// extractVersion pulls version number from UA string
func extractVersion(ua, prefix string) string {
	idx := strings.Index(ua, prefix)
	if idx == -1 {
		return ""
	}
	rest := ua[idx+len(prefix):]
	end := strings.IndexAny(rest, " ;)")
	if end == -1 {
		return rest
	}
	return rest[:end]
}
