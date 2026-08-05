package main

import (
	"errors"
	"strings"

	"github.com/goccy/go-yaml"
)

func generateMihomoConfig(config *MihomoConfig, peer *Peer, serverPubKey, endpoint string, iface map[string]string) (string, error) {
	proxies := config.Proxies
	if len(proxies) == 0 {
		return "", errors.New("empty proxies")
	}
	proxyGroup := config.ProxyGroups
	if len(proxyGroup) == 0 {
		return "", errors.New("empty proxy groups")
	}

	proxy := &proxies[0]
	proxy.Name = shortPeerName(peer.Name)
	proxy.PrivateKey = peer.PrivateKey
	proxy.PublicKey = serverPubKey

	serverIP, serverPort, _ := strings.Cut(endpoint, ":")
	proxy.Server = serverIP
	proxy.Port = serverPort

	proxy.IP = peer.IP

	// AmneziaWG params
	params := proxy.AmneziaWGOption
	params.JC = iface["Jc"]
	params.JMin = iface["Jmin"]
	params.JMax = iface["Jmax"]

	params.S1 = iface["S1"]
	params.S2 = iface["S2"]
	params.S3 = iface["S3"]
	params.S4 = iface["S4"]

	params.H1 = iface["H1"]
	params.H2 = iface["H2"]
	params.H3 = iface["H3"]
	params.H4 = iface["H4"]

	params.I1 = iface["I1"]
	params.I2 = iface["I2"]
	params.I3 = iface["I3"]
	params.I4 = iface["I4"]

	// AWG 3.0 params
	params.HeaderProtectionKey = iface["HeaderProtectionKey"]
	params.ContentPaddingAddition = iface["ContentPaddingAddition"]
	params.RekeyAfterTime = iface["RekeyAfterTime"]
	params.RekeyTimeout = iface["RekeyTimeout"]
	params.RejectAfterTime = iface["RejectAfterTime"]
	params.KeepaliveTimeout = iface["KeepaliveTimeout"]
	params.MaxHandshakeAttempts = iface["MaxHandshakeAttempts"]

	if isNotEmpty(params.HeaderProtectionKey, params.ContentPaddingAddition,
		params.RekeyAfterTime, params.RekeyTimeout,
		params.RejectAfterTime, params.KeepaliveTimeout,
		params.MaxHandshakeAttempts) {
		params.Version = "3"
	}

	for i := range config.ProxyGroups {
		config.ProxyGroups[i].Proxies = []string{proxy.Name}
	}

	data, err := yaml.Marshal(config)
	return string(data), err
}

func isNotEmpty(strs ...string) bool {
	for _, s := range strs {
		if s == "" {
			return false
		}
	}
	return true
}
