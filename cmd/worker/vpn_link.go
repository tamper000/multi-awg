package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	ini "gopkg.in/ini.v1"
)

type addressParts struct {
	clientIP   string
	clientIPv6 string
}

type dnsParts struct {
	dns1 string
	dns2 string
}

func parseAddresses(addressStr string) addressParts {
	addrs := strings.Split(addressStr, ",")
	clientIP := strings.Split(strings.TrimSpace(addrs[0]), "/")[0]

	clientIPv6 := ""
	if len(addrs) > 1 {
		clientIPv6 = strings.Split(strings.TrimSpace(addrs[1]), "/")[0]
	}

	return addressParts{clientIP: clientIP, clientIPv6: clientIPv6}
}

func parseDNS(dnsStr string) dnsParts {
	parts := strings.Split(dnsStr, ",")
	dns1 := strings.TrimSpace(parts[0])

	dns2 := dns1
	if len(parts) > 1 {
		dns2 = strings.TrimSpace(parts[1])
	}

	return dnsParts{dns1: dns1, dns2: dns2}
}

func buildInnerJSON(
	iface, peer *ini.Section,
	host string,
	port int,
	privateKey, peerPublicKey string,
	addrs addressParts,
	rawConfig []byte,
) map[string]any {
	inner := map[string]any{
		"H1": iface.Key("H1").String(), "H2": iface.Key("H2").String(),
		"H3": iface.Key("H3").String(), "H4": iface.Key("H4").String(),
		"Jc": iface.Key("Jc").String(), "Jmin": iface.Key("Jmin").String(),
		"Jmax": iface.Key("Jmax").String(),
		"S1":   iface.Key("S1").String(), "S2": iface.Key("S2").String(),
		"S3": iface.Key("S3").String(), "S4": iface.Key("S4").String(),
		"allowed_ips":           splitTrim(peer.Key("AllowedIPs").String()),
		"client_ip":             addrs.clientIP,
		"client_ipv6":           addrs.clientIPv6,
		"client_priv_key":       privateKey,
		"config":                string(rawConfig),
		"hostName":              host,
		"mtu":                   iface.Key("MTU").String(),
		"persistent_keep_alive": peer.Key("PersistentKeepalive").String(),
		"port":                  port,
		"server_pub_key":        peerPublicKey,
	}

	if hasAny(iface, "I1", "I2", "I3", "I4", "I5") {
		for i := 1; i <= 5; i++ {
			inner[fmt.Sprintf("I%d", i)] = iface.Key(fmt.Sprintf("I%d", i)).String()
		}
	}

	if psk := peer.Key("PresharedKey").String(); psk != "" {
		inner["psk_key"] = psk
	}

	return inner
}

func GenerateVPNURI(rawConfig []byte, description string) (string, error) {
	cfg, err := ini.Load(rawConfig)
	if err != nil {
		return "", fmt.Errorf("parse config: %w", err)
	}

	iface := cfg.Section("Interface")
	peer := cfg.Section("Peer")

	privateKey := iface.Key("PrivateKey").String()
	peerPublicKey := peer.Key("PublicKey").String()
	endpoint := peer.Key("Endpoint").String()

	if privateKey == "" || peerPublicKey == "" || endpoint == "" {
		return "", fmt.Errorf("missing required fields")
	}

	host, portStr, err := net.SplitHostPort(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint: %w", err)
	}

	port, _ := strconv.Atoi(portStr)

	addrs := parseAddresses(iface.Key("Address").String())
	dns := parseDNS(iface.Key("DNS").String())

	inner := buildInnerJSON(iface, peer, host, port, privateKey, peerPublicKey, addrs, rawConfig)
	innerJSON, _ := json.Marshal(inner)

	outer := map[string]any{
		"containers": []map[string]any{{
			"awg": map[string]any{
				"isThirdPartyConfig": true,
				"last_config":        string(innerJSON),
				"port":               portStr,
				"protocol_version":   "2",
				"transport_proto":    "udp",
			},
			"container": "amnezia-awg",
		}},
		"defaultContainer": "amnezia-awg",
		"description":      description,
		"dns1":             dns.dns1,
		"dns2":             dns.dns2,
		"hostName":         host,
	}

	outerJSON, _ := json.Marshal(outer)

	var zbuf bytes.Buffer
	zw := zlib.NewWriter(&zbuf)
	_, _ = zw.Write(outerJSON)
	_ = zw.Close()

	payload := make([]byte, 4+zbuf.Len())
	binary.BigEndian.PutUint32(payload[:4], uint32(len(outerJSON)))
	copy(payload[4:], zbuf.Bytes())

	return "vpn://" + base64.RawURLEncoding.EncodeToString(payload), nil
}

func splitTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))

	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}

	return result
}

func hasAny(section *ini.Section, keys ...string) bool {
	for _, k := range keys {
		if section.Key(k).String() != "" {
			return true
		}
	}

	return false
}
