package main

import (
	"errors"
	"strings"

	"github.com/goccy/go-yaml"
)

func generateMihomoConfig(config *MihomoConfig, peers []Peer, serverPubKey, endpoint string, iface map[string]string) (string, error) {
	if len(config.Proxies) == 0 && len(peers) > 0 {
		return "", errors.New("empty proxies")
	}
	if len(config.ProxyGroups) == 0 && len(peers) > 0 {
		return "", errors.New("empty proxy groups")
	}
	if len(peers) == 0 {
		config.Proxies = []Proxy{{Name: "Нету конфигов", Type: "direct"}}
		if len(config.ProxyGroups) == 0 {
			config.ProxyGroups = []ProxyGroup{{Name: "🌐 Обход РФ", Type: "select"}}
		}
		for i := range config.ProxyGroups {
			config.ProxyGroups[i].Proxies = []string{"Нету конфигов"}
		}
		data, err := yaml.Marshal(config)
		return string(data), err
	}

	base := config.Proxies[0]
	proxies := make([]Proxy, 0, len(peers))
	proxyNames := make([]string, 0, len(peers))
	serverIP, serverPort, _ := strings.Cut(endpoint, ":")
	for _, peer := range peers {
		proxy := base
		proxy.Name = peer.Name
		proxy.PrivateKey = peer.PrivateKey
		proxy.PublicKey = serverPubKey
		proxy.Server = serverIP
		proxy.Port = serverPort
		proxy.IP = peer.IP
		proxies = append(proxies, proxy)
		proxyNames = append(proxyNames, proxy.Name)
	}
	config.Proxies = proxies

	// AmneziaWGOption хранится указателем, поэтому эти параметры выставляются
	// через общий объект и применяются ко всем прокси.
	params := config.Proxies[0].AmneziaWGOption
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
		config.ProxyGroups[i].Proxies = proxyNames
	}

	data, err := yaml.Marshal(config)
	return string(data), err
}

func isNotEmpty(strs ...string) bool {
	for _, s := range strs {
		if s != "" {
			return true
		}
	}

	return false
}
