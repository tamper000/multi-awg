package main

import "time"

const (
	dnsDefault = "94.140.14.14"
	mtuDefault = 1420
)

type Peer struct {
	Name       string    `json:"name"`
	PrivateKey string    `json:"private_key"`
	PublicKey  string    `json:"public_key"`
	IP         string    `json:"ip"`
	DNS        string    `json:"dns"`
	CreatedAt  time.Time `json:"created_at"`
}

// SnifferConfig представляет конфигурацию сниффера
type SnifferConfig struct {
	Enable              bool `yaml:"enable,omitempty"`
	ForceDNSMapping     bool `yaml:"force-dns-mapping,omitempty"`
	ParsePureIP         bool `yaml:"parse-pure-ip,omitempty"`
	OverrideDestination bool `yaml:"override-destination,omitempty"`
	Sniff               struct {
		HTTP struct {
			Ports               []string `yaml:"ports,omitempty"`
			OverrideDestination bool     `yaml:"override-destination,omitempty"`
		} `yaml:"HTTP,omitempty"`
		TLS struct {
			Ports []string `yaml:"ports,omitempty"`
		} `yaml:"TLS,omitempty"`
		QUIC struct {
			Ports []string `yaml:"ports,omitempty"`
		} `yaml:"QUIC,omitempty"`
	} `yaml:"sniff,omitempty"`
	ForceDomain    []string `yaml:"force-domain,omitempty"`
	SkipDomain     []string `yaml:"skip-domain,omitempty"`
	SkipSrcAddress []string `yaml:"skip-src-address,omitempty"`
	SkipDstAddress []string `yaml:"skip-dst-address,omitempty"`
}

type DNSConfig struct {
	Enable                       bool              `yaml:"enable,omitempty"`
	CacheAlgorithm              string            `yaml:"cache-algorithm,omitempty"`
	PreferH3                    bool              `yaml:"prefer-h3,omitempty"`
	UseHosts                    bool              `yaml:"use-hosts,omitempty"`
	UseSystemHosts              bool              `yaml:"use-system-hosts,omitempty"`
	RespectRules                bool              `yaml:"respect-rules,omitempty"`
	Listen                      string            `yaml:"listen,omitempty"`
	IPv6                        bool              `yaml:"ipv6,omitempty"`
	DefaultNameserver           []string          `yaml:"default-nameserver,omitempty"`
	EnhancedMode                string            `yaml:"enhanced-mode,omitempty"`
	FakeIPRange                string            `yaml:"fake-ip-range,omitempty"`
	FakeIPFilterMode            string            `yaml:"fake-ip-filter-mode,omitempty"`
	FakeIPFilter                []string          `yaml:"fake-ip-filter,omitempty"`
	FakeIPTTL                   int               `yaml:"fake-ip-ttl,omitempty"`
	NameserverPolicy             map[string]any    `yaml:"nameserver-policy,omitempty"`
	Nameserver                  []string          `yaml:"nameserver,omitempty"`
	Fallback                    []string          `yaml:"fallback,omitempty"`
	ProxyServerNameserver       []string          `yaml:"proxy-server-nameserver,omitempty"`
	ProxyServerNameserverPolicy map[string]string `yaml:"proxy-server-nameserver-policy,omitempty"`
	DirectNameserver            []string          `yaml:"direct-nameserver,omitempty"`
	DirectNameserverFollowPolicy bool              `yaml:"direct-nameserver-follow-policy,omitempty"`
	FallbackFilter               FallbackFilter    `yaml:"fallback-filter,omitempty"`
}

type FallbackFilter struct {
	GeoIP     bool     `yaml:"geoip,omitempty"`
	GeoIPCode string   `yaml:"geoip-code,omitempty"`
	Geosite   []string `yaml:"geosite,omitempty"`
	IPCIDR    []string `yaml:"ipcidr,omitempty"`
	Domain    []string `yaml:"domain,omitempty"`
}

// AmneziaWGOption представляет опции Amnezia WG
type AmneziaWGOption struct {
	JC                     string `yaml:"jc,omitempty"`
	JMin                   string `yaml:"jmin,omitempty"`
	JMax                   string `yaml:"jmax,omitempty"`
	S1                     string `yaml:"s1,omitempty"`
	S2                     string `yaml:"s2,omitempty"`
	S3                     string `yaml:"s3,omitempty"`
	S4                     string `yaml:"s4,omitempty"`
	H1                     string `yaml:"h1,omitempty"`
	H2                     string `yaml:"h2,omitempty"`
	H3                     string `yaml:"h3,omitempty"`
	H4                     string `yaml:"h4,omitempty"`
	I1                     string `yaml:"i1,omitempty"`
	I2                     string `yaml:"i2,omitempty"`
	I3                     string `yaml:"i3,omitempty"`
	I4                     string `yaml:"i4,omitempty"`
	Version                string `yaml:"version,omitempty"`
	HeaderProtectionKey    string `yaml:"header-protection-key,omitempty"`
	ContentPaddingAddition string `yaml:"content-padding-addition,omitempty"`
	RekeyAfterTime         string `yaml:"rekey-after-time,omitempty"`
	RekeyTimeout           string `yaml:"rekey-timeout,omitempty"`
	RejectAfterTime        string `yaml:"reject-after-time,omitempty"`
	KeepaliveTimeout       string `yaml:"keepalive-timeout,omitempty"`
	MaxHandshakeAttempts   string `yaml:"max-handshake-attempts,omitempty"`
}

// Proxy представляет конфигурацию прокси
type Proxy struct {
	Name                string           `yaml:"name,omitempty"`
	Type                string           `yaml:"type,omitempty"`
	PrivateKey          string           `yaml:"private-key,omitempty"`
	Server              string           `yaml:"server,omitempty"`
	Port                string           `yaml:"port,omitempty"`
	IP                  string           `yaml:"ip,omitempty"`
	PublicKey           string           `yaml:"public-key,omitempty"`
	AllowedIPs          []string         `yaml:"allowed-ips,omitempty"`
	PersistentKeepalive int              `yaml:"persistent-keepalive,omitempty"`
	MTU                 int              `yaml:"mtu,omitempty"`
	AmneziaWGOption     *AmneziaWGOption `yaml:"amnezia-wg-option,omitempty"`
}

// ProxyGroup представляет группу прокси
type ProxyGroup struct {
	Name    string   `yaml:"name,omitempty"`
	Type    string   `yaml:"type,omitempty"`
	Proxies []string `yaml:"proxies,omitempty"`
}

// MihomoConfig представляет полную конфигурацию
type MihomoConfig struct {
	Sniffer     SnifferConfig `yaml:"sniffer,omitempty"`
	DNS         DNSConfig     `yaml:"dns,omitempty"`
	Proxies     []Proxy       `yaml:"proxies,omitempty"`
	ProxyGroups []ProxyGroup  `yaml:"proxy-groups,omitempty"`
	Rules       []string      `yaml:"rules,omitempty"`
}
