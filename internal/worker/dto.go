package worker

type Peer struct {
	Name      string `json:"name"`
	IP        string `json:"ip"`
	PublicKey string `json:"public_key"`
	Config    string `json:"config"`
}

type PeerConfig struct {
	Config string `json:"config"`
}

type Sub struct {
	Conf       string `json:"conf"`
	VPNLink    string `json:"vpn_link"`
	MihomoYaml string `json:"mihomo_yaml"`
}

type SubPeer struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

type Stats struct {
	Name          string `json:"name"`
	IP            string `json:"ip"`
	Received      int64  `json:"received"`
	Sent          int64  `json:"sent"`
	LastHandshake int64  `json:"last_handshake"`
}

type Error struct {
	Error string `json:"error"`
}
