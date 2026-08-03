package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/curve25519"
)

const peersFile = "peers.json"

// --- WireGuard config parsing ---

type wgPeer struct {
	PublicKey  string
	AllowedIPs string
}

// parseWGConfig reads the raw config and extracts [Interface] key-value pairs
// and [Peer] blocks as raw text.
func parseWGConfig(confPath string) (iface map[string]string, peerBlocks []string, raw string, err error) {
	data, err := os.ReadFile(confPath)
	if err != nil {
		return nil, nil, "", err
	}
	raw = string(data)

	iface = make(map[string]string)
	var currentSection string

	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[Interface]" {
			currentSection = "Interface"
			continue
		}
		if trimmed == "[Peer]" {
			currentSection = "Peer"
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			currentSection = ""
			continue
		}

		if currentSection == "Interface" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, ";") {
			if k, v, ok := parseKV(trimmed); ok {
				iface[k] = v
			}
		}
	}

	peerBlocks = extractPeerBlocks(raw)
	return
}

// extractPeerBlocks returns each [Peer] ... section as a separate string.
func extractPeerBlocks(raw string) []string {
	var blocks []string
	var current []string
	inPeer := false

	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[Peer]" {
			if inPeer && len(current) > 0 {
				blocks = append(blocks, strings.Join(current, "\n"))
			}
			current = []string{line}
			inPeer = true
			continue
		}
		if inPeer {
			if strings.HasPrefix(trimmed, "[") {
				blocks = append(blocks, strings.Join(current, "\n"))
				current = nil
				inPeer = false
			} else {
				current = append(current, line)
			}
		}
	}
	if inPeer && len(current) > 0 {
		blocks = append(blocks, strings.Join(current, "\n"))
	}
	return blocks
}

// parsePeerBlock extracts PublicKey and AllowedIPs from a [Peer] text block.
func parsePeerBlock(block string) wgPeer {
	var p wgPeer
	for _, line := range strings.Split(block, "\n") {
		if k, v, ok := parseKV(strings.TrimSpace(line)); ok {
			switch k {
			case "PublicKey":
				p.PublicKey = v
			case "AllowedIPs":
				p.AllowedIPs = v
			}
		}
	}
	return p
}

// parseKV parses "Key = Value" or "Key=Value" lines.
func parseKV(line string) (key, value string, ok bool) {
	if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
		return "", "", false
	}
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

// peerSectionKey extracts the IP from AllowedIPs ("10.0.0.2/32" -> "10.0.0.2").
func peerIP(allowedIPs string) string {
	if idx := strings.Index(allowedIPs, "/"); idx != -1 {
		return allowedIPs[:idx]
	}
	return allowedIPs
}

// --- Statistics ---

type peerStats struct {
	LastHandshake int64
	RxBytes       int64
	TxBytes       int64
}

// parseWGShowDump parses "awg show all dump" tab-separated output.
// Line fields: iface, pubkey, preshared, endpoint, allowed-ips, handshake(unix), rx, tx, keepalive.
func parseWGShowDump(raw string) map[string]peerStats {
	stats := make(map[string]peerStats)
	for _, line := range strings.Split(raw, "\n") {
		f := strings.Split(line, "\t")
		if len(f) < 8 {
			continue
		}
		handshake, _ := strconv.ParseInt(f[5], 10, 64)
		rx, _ := strconv.ParseInt(f[6], 10, 64)
		tx, _ := strconv.ParseInt(f[7], 10, 64)
		stats[f[1]] = peerStats{LastHandshake: handshake, RxBytes: rx, TxBytes: tx}
	}
	return stats
}

// --- IP allocation ---

func parseSubnet(confPath string) (first, last net.IP, err error) {
	iface, _, _, err := parseWGConfig(confPath)
	if err != nil {
		return nil, nil, err
	}
	addr := iface["Address"]
	if addr == "" {
		return nil, nil, fmt.Errorf("Address not found in [Interface]")
	}

	ip, ipnet, err := net.ParseCIDR(addr)
	if err != nil {
		ip = net.ParseIP(addr)
		if ip == nil {
			return nil, nil, fmt.Errorf("parse address: %s", addr)
		}
		mask := net.CIDRMask(24, 32)
		ipnet = &net.IPNet{IP: ip.Mask(mask), Mask: mask}
	}

	first = ip.Mask(ipnet.Mask)
	last = make(net.IP, len(first))
	copy(last, first)
	for i := range last {
		last[i] |= ^ipnet.Mask[i]
	}
	first[3]++
	last[3]--
	return first, last, nil
}

func nextAvailableIP(confPath string, peers []Peer) (string, error) {
	first, last, err := parseSubnet(confPath)
	if err != nil {
		return "", err
	}

	used := make(map[string]bool)
	used[first.String()] = true

	// collect IPs from existing [Peer] blocks in config
	_, peerBlocks, _, err := parseWGConfig(confPath)
	if err == nil {
		for _, block := range peerBlocks {
			p := parsePeerBlock(block)
			if p.AllowedIPs != "" {
				used[peerIP(p.AllowedIPs)] = true
			}
		}
	}

	// collect IPs from peers.json
	for _, p := range peers {
		used[p.IP] = true
	}

	ip := make(net.IP, len(first))
	copy(ip, first)
	for {
		if !used[ip.String()] {
			return ip.String(), nil
		}
		if ip.Equal(last) {
			return "", fmt.Errorf("no available IPs in subnet")
		}
		for i := len(ip) - 1; i >= 0; i-- {
			ip[i]++
			if ip[i] != 0 {
				break
			}
		}
	}
}

// --- Key generation ---

func generateKeyPair() (privateKey, publicKey string, err error) {
	priv := make([]byte, 32)
	if _, err = rand.Read(priv); err != nil {
		return "", "", fmt.Errorf("generate private key: %w", err)
	}
	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		return "", "", fmt.Errorf("derive public key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(priv), base64.StdEncoding.EncodeToString(pub), nil
}

func extractServerPublicKey(confPath string) (string, error) {
	iface, _, _, err := parseWGConfig(confPath)
	if err != nil {
		return "", err
	}
	privKey := iface["PrivateKey"]
	if privKey == "" {
		return "", fmt.Errorf("PrivateKey not found in [Interface]")
	}
	privBytes, err := base64.StdEncoding.DecodeString(privKey)
	if err != nil {
		return "", fmt.Errorf("decode private key: %w", err)
	}
	pub, err := curve25519.X25519(privBytes, curve25519.Basepoint)
	if err != nil {
		return "", fmt.Errorf("derive public key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(pub), nil
}

func extractAmneziaParams(confPath string) (string, error) {
	iface, _, _, err := parseWGConfig(confPath)
	if err != nil {
		return "", err
	}

	amneziaKeys := []string{
		"H1", "H2", "H3", "H4", "H5",
		"S1", "S2", "S3", "S4",
		"Jc", "Jmin", "Jmax",
		"I1", "I2", "I3", "I4", "I5",
		"HeaderProtectionKey", "ContentPaddingAddition",
		"RekeyAfterTime", "RekeyTimeout", "RejectAfterTime",
		"KeepaliveTimeout", "MaxHandshakeAttempts",
	}

	var params []string
	for _, k := range amneziaKeys {
		if v, ok := iface[k]; ok && v != "" {
			params = append(params, fmt.Sprintf("%s = %s", k, v))
		}
	}
	if len(params) == 0 {
		return "", nil
	}
	return strings.Join(params, "\n") + "\n", nil
}

// --- Config mutation ---

// appendPeerToConfig adds a new [Peer] section at the end of the file.
func appendPeerToConfig(confPath, publicKey, ip string) error {
	data, err := os.ReadFile(confPath)
	if err != nil {
		return err
	}

	peer := fmt.Sprintf("\n[Peer]\nPublicKey = %s\nAllowedIPs = %s/32\n", publicKey, ip)
	return os.WriteFile(confPath, append(data, []byte(peer)...), 0644)
}

// removePeerFromConfig removes the [Peer] block containing the given public key.
func removePeerFromConfig(confPath, publicKey string) error {
	data, err := os.ReadFile(confPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	var result []string
	i := 0

	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])

		if trimmed == "[Peer]" {
			// peek ahead to find PublicKey in this block
			peerEnd := i + 1
			match := false
			for peerEnd < len(lines) {
				ahead := strings.TrimSpace(lines[peerEnd])
				if strings.HasPrefix(ahead, "[") {
					break
				}
				if k, v, ok := parseKV(ahead); ok && k == "PublicKey" && v == publicKey {
					match = true
				}
				peerEnd++
			}

			if match {
				// skip this entire block + trailing blank line
				i = peerEnd
				if i < len(lines) && strings.TrimSpace(lines[i]) == "" {
					i++
				}
				continue
			}
		}

		result = append(result, lines[i])
		i++
	}

	return os.WriteFile(confPath, []byte(strings.Join(result, "\n")), 0644)
}

// --- Client config generation ---

func generateClientConfig(privateKey, serverPubKey, ip, dns, endpoint, amneziaParams string) string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	b.WriteString(fmt.Sprintf("PrivateKey = %s\n", privateKey))
	b.WriteString(fmt.Sprintf("Address = %s/32\n", ip))
	b.WriteString(fmt.Sprintf("DNS = %s\n", dns))
	b.WriteString(fmt.Sprintf("MTU = %d\n", mtuDefault))
	b.WriteString("\n")
	if amneziaParams != "" {
		b.WriteString(amneziaParams)
		b.WriteString("\n")
	}
	b.WriteString("[Peer]\n")
	b.WriteString(fmt.Sprintf("PublicKey = %s\n", serverPubKey))
	if endpoint != "" {
		b.WriteString(fmt.Sprintf("Endpoint = %s\n", endpoint))
	}
	b.WriteString("AllowedIPs = 0.0.0.0/0, ::/0\n")
	b.WriteString("PersistentKeepalive = 25\n")
	return b.String()
}

// --- peers.json CRUD ---

func loadPeers(dir string) ([]Peer, error) {
	data, err := os.ReadFile(filepath.Join(dir, peersFile))
	if os.IsNotExist(err) {
		return []Peer{}, nil
	}
	if err != nil {
		return nil, err
	}
	var peers []Peer
	if err := json.Unmarshal(data, &peers); err != nil {
		return nil, err
	}
	return peers, nil
}

func savePeers(dir string, peers []Peer) error {
	data, err := json.MarshalIndent(peers, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, peersFile), data, 0644)
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
