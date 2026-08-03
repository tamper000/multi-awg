package main

import (
	"fmt"
	"net"
	"strings"
)

var mihomoKeyConf = map[string]string{
	"jc": "Jc", "jmin": "Jmin", "jmax": "Jmax",
	"s1": "S1", "s2": "S2", "s3": "S3", "s4": "S4",
	"h1": "H1", "h2": "H2", "h3": "H3", "h4": "H4",
	"i1": "I1", "i2": "I2", "i3": "I3", "i4": "I4", "i5": "I5",
}

var mihomoKeyOrder = []string{
	"jc", "jmin", "jmax",
	"s1", "s2", "s3", "s4",
	"h1", "h2", "h3", "h4",
	"i1", "i2", "i3", "i4", "i5",
}

func generateMihomoConfig(tmpl, name string, peer *Peer, serverPubKey, endpoint string, iface map[string]string) (string, error) {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint: %w", err)
	}

	out := strings.NewReplacer(
		"nameConfig", name,
		"private_key", peer.PrivateKey,
		"server_ip", host,
		"server_port", port,
		"client_ip", peer.IP,
		"public_key", serverPubKey,
	).Replace(tmpl)

	var b strings.Builder
	var blockIndent string
	present := map[string]bool{}

	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if trimmed == "amnezia-wg-option:" {
			blockIndent = line[:len(line)-len(trimmed)]
			b.WriteString(line + "\n")
			continue
		}

		matched := false
		for _, k := range mihomoKeyOrder {
			if !strings.HasPrefix(trimmed, k+":") {
				continue
			}
			matched = true
			if v, ok := iface[mihomoKeyConf[k]]; ok && v != "" {
				present[k] = true
				indent := line[:len(line)-len(trimmed)]
				b.WriteString(indent + k + ": " + v + "\n")
			}
			break
		}
		if !matched {
			b.WriteString(line + "\n")
		}
	}

	if blockIndent != "" {
		for _, k := range mihomoKeyOrder {
			if present[k] {
				continue
			}
			if v, ok := iface[mihomoKeyConf[k]]; ok && v != "" {
				b.WriteString(blockIndent + "  " + k + ": " + v + "\n")
			}
		}
	}

	return b.String(), nil
}
