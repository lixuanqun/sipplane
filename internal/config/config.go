package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is data-plane bootstrap configuration (RFC 0004).
type Config struct {
	Listen         string `yaml:"listen"`
	Transport      string `yaml:"transport"` // udp, tcp, tls, or comma-separated
	AdvertisedHost string `yaml:"advertised_host"`
	AdvertisedPort int    `yaml:"advertised_port"`
	HTTPListen     string `yaml:"http_listen"`
	ConfigDir      string `yaml:"config_dir"`
	Realm          string `yaml:"realm"`
	LogLevel       string `yaml:"log_level"`
	ControlURL     string `yaml:"control_url"` // optional P2a: watch control plane
	ControlToken   string `yaml:"control_token"` // Bearer token for control plane (DP watcher)
	RedisAddr      string `yaml:"redis_addr"`  // optional P3: redis location
	ConfigStaleAfter string `yaml:"config_stale_after"`
	TLSCertFile    string `yaml:"tls_cert_file"`
	TLSKeyFile     string `yaml:"tls_key_file"`
	EnablePath     bool   `yaml:"enable_path"` // RFC 3327 Path on REGISTER 200
	EnableOutbound bool   `yaml:"enable_outbound"` // RFC 5626
	OutboundSecret string `yaml:"outbound_secret"`
	HEPAddr        string `yaml:"hep_addr"` // e.g. 127.0.0.1:9060 Homer HEP
	HEPCaptureID   uint32 `yaml:"hep_capture_id"`
	OTelEndpoint   string `yaml:"otel_endpoint"` // e.g. http://localhost:4318
	RedirectPolicy string `yaml:"redirect_policy"` // follow | passthrough | reject (default passthrough)
	Policies       PoliciesConfig `yaml:"policies"`
	// AllowMissingAdvertised permits empty advertised_host when listen is loopback-only.
	AllowLoopbackWithoutAdvertised bool `yaml:"-"`
}

// PoliciesConfig is optional ingress ACL / rate-limit (P2b).
type PoliciesConfig struct {
	ACL       *ACLConfig       `yaml:"acl"`
	RateLimit *RateLimitConfig `yaml:"rateLimit"`
}

// ACLConfig mirrors policy.ACL YAML fields.
type ACLConfig struct {
	AllowCIDRs []string `yaml:"allowCidrs"`
	DenyCIDRs  []string `yaml:"denyCidrs"`
	Methods    []string `yaml:"methods"`
}

// RateLimitConfig mirrors policy.RateLimit YAML fields.
type RateLimitConfig struct {
	CPS     float64 `yaml:"cps"`
	Burst   int     `yaml:"burst"`
	Backend string  `yaml:"backend"` // local | redis | auto (default: redis if redis_addr set)
	Key     string  `yaml:"key"`     // global | ip (default global)
}

// Default returns sensible lab defaults.
func Default() Config {
	return Config{
		Listen:         "0.0.0.0:5060",
		Transport:      "udp",
		AdvertisedPort: 5060,
		HTTPListen:     "0.0.0.0:8080",
		ConfigDir:      "examples/config",
		Realm:          "sipplane",
		LogLevel:       "info",
	}
}

// LoadFile loads bootstrap YAML and applies defaults.
func LoadFile(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) applyDefaults() {
	d := Default()
	if c.Listen == "" {
		c.Listen = d.Listen
	}
	if c.Transport == "" {
		c.Transport = d.Transport
	}
	if c.HTTPListen == "" {
		c.HTTPListen = d.HTTPListen
	}
	if c.AdvertisedPort == 0 {
		_, portStr, err := net.SplitHostPort(c.Listen)
		if err == nil {
			if p, err := strconv.Atoi(portStr); err == nil {
				c.AdvertisedPort = p
			}
		}
		if c.AdvertisedPort == 0 {
			c.AdvertisedPort = 5060
		}
	}
	if c.Realm == "" {
		c.Realm = d.Realm
	}
	if c.LogLevel == "" {
		c.LogLevel = d.LogLevel
	}
}

// Validate enforces RFC 0004 advertised_host rules.
func (c Config) Validate() error {
	c.applyDefaults()
	host, _, err := net.SplitHostPort(c.Listen)
	if err != nil {
		// allow host without port for flexibility
		host = c.Listen
	}
	loopback := host == "127.0.0.1" || host == "::1" || host == "localhost"
	if strings.TrimSpace(c.AdvertisedHost) == "" {
		if loopback || c.AllowLoopbackWithoutAdvertised {
			return nil
		}
		return fmt.Errorf("advertised_host is required when listen is not loopback (RFC 0004); refusing to start with ephemeral/pod IP risk")
	}
	return nil
}

// AdvertisedSIPURI returns sip:host:port;lr for Record-Route.
func (c Config) AdvertisedSIPURI() string {
	host := c.AdvertisedHost
	if host == "" {
		host = "127.0.0.1"
	}
	port := c.AdvertisedPort
	if port == 0 {
		port = 5060
	}
	return fmt.Sprintf("sip:%s:%d;lr", host, port)
}

// Transports returns normalized transport list.
func (c Config) Transports() []string {
	parts := strings.Split(c.Transport, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return []string{"udp"}
	}
	return out
}

// ApplyFromEnv overlays SIPPLANE_* environment variables.
func (c *Config) ApplyFromEnv() {
	if v := os.Getenv("SIPPLANE_LISTEN"); v != "" {
		c.Listen = v
	}
	if v := os.Getenv("SIPPLANE_ADVERTISED_HOST"); v != "" {
		c.AdvertisedHost = v
	}
	if v := os.Getenv("SIPPLANE_ADVERTISED_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.AdvertisedPort = p
		}
	}
	if v := os.Getenv("SIPPLANE_HTTP_LISTEN"); v != "" {
		c.HTTPListen = v
	}
	if v := os.Getenv("SIPPLANE_CONFIG_DIR"); v != "" {
		c.ConfigDir = v
	}
	if v := os.Getenv("SIPPLANE_REALM"); v != "" {
		c.Realm = v
	}
	if v := os.Getenv("SIPPLANE_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("SIPPLANE_TRANSPORT"); v != "" {
		c.Transport = v
	}
	if v := os.Getenv("SIPPLANE_CONTROL_URL"); v != "" {
		c.ControlURL = v
	}
	if v := os.Getenv("SIPPLANE_CONTROL_TOKEN"); v != "" {
		c.ControlToken = v
	}
	if v := os.Getenv("SIPPLANE_REDIS_ADDR"); v != "" {
		c.RedisAddr = v
	}
	if v := os.Getenv("SIPPLANE_HEP_ADDR"); v != "" {
		c.HEPAddr = v
	}
	if v := os.Getenv("SIPPLANE_OTEL_ENDPOINT"); v != "" {
		c.OTelEndpoint = v
	}
	if v := os.Getenv("SIPPLANE_ENABLE_OUTBOUND"); v == "1" || strings.EqualFold(v, "true") {
		c.EnableOutbound = true
	}
}

// HasPolicies reports whether any ingress policy is configured.
func (c Config) HasPolicies() bool {
	if c.Policies.ACL != nil {
		a := c.Policies.ACL
		if len(a.AllowCIDRs) > 0 || len(a.DenyCIDRs) > 0 || len(a.Methods) > 0 {
			return true
		}
	}
	if c.Policies.RateLimit != nil && c.Policies.RateLimit.CPS > 0 {
		return true
	}
	return false
}
