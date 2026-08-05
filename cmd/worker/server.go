package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/goccy/go-yaml"
)

type Server struct {
	cli            *client.Client
	containerName  string
	confDir        string
	token          string
	serverEndpoint string
	mihomoTemplate string
	mu             sync.RWMutex
}

type Config struct {
	ContainerName  string
	ConfDir        string
	Token          string
	ServerEndpoint string
	MihomoTemplate string
}

func NewServer(cli *client.Client, cfg Config) *Server {
	return &Server{
		cli:            cli,
		containerName:  cfg.ContainerName,
		confDir:        cfg.ConfDir,
		token:          cfg.Token,
		serverEndpoint: cfg.ServerEndpoint,
		mihomoTemplate: cfg.MihomoTemplate,
	}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{
		Logger:  slog.NewLogLogger(slog.Default().Handler(), slog.LevelInfo),
		NoColor: true,
	}))
	r.Use(middleware.Recoverer)

	r.Get("/api/health", s.health)

	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Post("/api/peers", s.createPeer)
		r.Get("/api/peers", s.listPeers)
		r.Delete("/api/peers", s.deletePeers)
		r.Post("/api/peers/freeze", s.freezePeers)
		r.Post("/api/peers/unfreeze", s.unfreezePeers)
		r.Get("/api/peers/{name}/config", s.getPeerConfig)
		r.Get("/api/peers/{name}/stats", s.getPeerStats)
		r.Get("/api/peers/{name}/sub", s.getPeerSub)
		r.Get("/api/stats", s.getStats)
		r.Post("/api/sync", s.handleSync)
	})

	return r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	peers, err := loadPeers(s.confDir)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	stats, err := s.fetchStats(r)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	type statResp struct {
		Name          string `json:"name"`
		IP            string `json:"ip"`
		Received      int64  `json:"received"`
		Sent          int64  `json:"sent"`
		LastHandshake int64  `json:"last_handshake"`
	}
	result := make([]statResp, 0, len(peers))
	for _, p := range peers {
		st := stats[p.PublicKey]
		result = append(result, statResp{
			Name:          p.Name,
			IP:            p.IP,
			Received:      st.RxBytes,
			Sent:          st.TxBytes,
			LastHandshake: st.LastHandshake,
		})
	}
	writeJSON(w, 200, result)
}

func (s *Server) getPeerStats(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	name := chi.URLParam(r, "name")

	peers, err := loadPeers(s.confDir)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	var peer *Peer
	for i := range peers {
		if peers[i].Name == name {
			peer = &peers[i]
			break
		}
	}
	if peer == nil {
		writeJSON(w, 404, map[string]string{"error": "peer not found"})
		return
	}

	stats, err := s.fetchStats(r)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	st := stats[peer.PublicKey]

	writeJSON(w, 200, map[string]interface{}{
		"name":           peer.Name,
		"ip":             peer.IP,
		"received":       st.RxBytes,
		"sent":           st.TxBytes,
		"last_handshake": st.LastHandshake,
	})
}

// fetchStats runs "awg show all dump" and returns per-public-key stats.
func (s *Server) fetchStats(r *http.Request) (map[string]peerStats, error) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	out, err := s.execOutput(ctx, "awg", "show", "all", "dump")
	if err != nil {
		return nil, fmt.Errorf("awg show: %w", err)
	}
	return parseWGShowDump(out), nil
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.token == "" {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+s.token {
			writeJSON(w, 401, map[string]string{"error": "unauthorized"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) createPeer(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var req struct {
		Name string `json:"name"`
		DNS  string `json:"dns"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, 400, map[string]string{"error": "name is required"})
		return
	}
	if req.DNS == "" {
		req.DNS = dnsDefault
	}

	confPath := s.confPath()

	privKey, pubKey, err := generateKeyPair()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	peers, err := loadPeers(s.confDir)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("load peers: %v", err)})
		return
	}

	for _, p := range peers {
		if p.Name == req.Name {
			writeJSON(w, 409, map[string]string{"error": "peer name already exists"})
			return
		}
	}

	ip, err := nextAvailableIP(confPath, peers)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	if err := appendPeerToConfig(confPath, pubKey, ip); err != nil {
		writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("update config: %v", err)})
		return
	}

	peer := Peer{
		Name:       req.Name,
		PrivateKey: privKey,
		PublicKey:  pubKey,
		IP:         ip,
		DNS:        req.DNS,
		CreatedAt:  nowUTC(),
	}
	peers = append(peers, peer)

	if err := savePeers(s.confDir, peers); err != nil {
		writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("save peers: %v", err)})
		return
	}

	if err := s.syncAWG(); err != nil {
		writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("sync awg: %v", err)})
		return
	}

	serverPubKey, _ := extractServerPublicKey(confPath)
	amneziaParams, _ := extractAmneziaParams(confPath)
	clientConf := generateClientConfig(privKey, serverPubKey, ip, req.DNS, s.serverEndpoint, amneziaParams)

	writeJSON(w, 201, map[string]interface{}{
		"name":       peer.Name,
		"ip":         peer.IP,
		"public_key": peer.PublicKey,
		"config":     clientConf,
	})
}

func (s *Server) listPeers(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	peers, err := loadPeers(s.confDir)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	type peerResp struct {
		Name      string    `json:"name"`
		IP        string    `json:"ip"`
		CreatedAt time.Time `json:"created_at"`
	}
	result := make([]peerResp, len(peers))
	for i, p := range peers {
		result[i] = peerResp{Name: p.Name, IP: p.IP, CreatedAt: p.CreatedAt}
	}
	writeJSON(w, 200, result)
}

func (s *Server) getPeerConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	name := chi.URLParam(r, "name")

	peers, err := loadPeers(s.confDir)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	var peer *Peer
	for i := range peers {
		if peers[i].Name == name {
			peer = &peers[i]
			break
		}
	}
	if peer == nil {
		writeJSON(w, 404, map[string]string{"error": "peer not found"})
		return
	}

	confPath := s.confPath()
	serverPubKey, err := extractServerPublicKey(confPath)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	amneziaParams, _ := extractAmneziaParams(confPath)
	clientConf := generateClientConfig(peer.PrivateKey, serverPubKey, peer.IP, peer.DNS, s.serverEndpoint, amneziaParams)

	writeJSON(w, 200, map[string]string{"config": clientConf})
}

func (s *Server) getPeerSub(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	name := chi.URLParam(r, "name")

	peers, err := loadPeers(s.confDir)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	var peer *Peer
	for i := range peers {
		if peers[i].Name == name {
			peer = &peers[i]
			break
		}
	}
	if peer == nil {
		writeJSON(w, 404, map[string]string{"error": "peer not found"})
		return
	}

	confPath := s.confPath()
	iface, _, _, err := parseWGConfig(confPath)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	serverPubKey, err := extractServerPublicKey(confPath)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	amneziaParams, _ := extractAmneziaParams(confPath)

	clientConf := generateClientConfig(peer.PrivateKey, serverPubKey, peer.IP, peer.DNS, s.serverEndpoint, amneziaParams)

	vpnLink, err := GenerateVPNURI([]byte(clientConf), shortPeerName(peer.Name))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	var mihomoYaml string
	tmpl, err := os.ReadFile(s.mihomoTemplate)
	if err != nil {
		slog.Error("read mihomo template", "err", err)
	} else {
		var mihomoCfg MihomoConfig
		if err := yaml.Unmarshal(tmpl, &mihomoCfg); err != nil {
			slog.Error("parse mihomo template", "err", err)
		} else {
			mihomoYaml, err = generateMihomoConfig(&mihomoCfg, peer, serverPubKey, s.serverEndpoint, iface)
			if err != nil {
				slog.Error("generate mihomo config", "err", err)
				mihomoYaml = ""
			}
		}
	}

	writeJSON(w, 200, map[string]string{
		"conf":        clientConf,
		"vpn_link":    vpnLink,
		"mihomo_yaml": mihomoYaml,
	})
}

func shortPeerName(name string) string {
	if _, after, ok := strings.Cut(name, "."); ok {
		return after
	}
	return name
}

func (s *Server) deletePeers(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Names []string `json:"names"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}
	if len(req.Names) == 0 {
		writeJSON(w, 400, map[string]string{"error": "names required"})
		return
	}
	s.removePeers(w, r, req.Names)
}

func (s *Server) removePeers(w http.ResponseWriter, r *http.Request, names []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	peers, err := loadPeers(s.confDir)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	remove := make(map[string]bool, len(names))
	for _, n := range names {
		remove[n] = true
	}

	var kept []Peer
	var pubKeys []string
	for _, p := range peers {
		if remove[p.Name] {
			pubKeys = append(pubKeys, p.PublicKey)
		} else {
			kept = append(kept, p)
		}
	}

	if len(pubKeys) == 0 {
		writeJSON(w, 404, map[string]string{"error": "peer not found"})
		return
	}

	for _, pubKey := range pubKeys {
		if err := removePeerFromConfig(s.confPath(), pubKey); err != nil {
			writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("update config: %v", err)})
			return
		}
	}
	if err := savePeers(s.confDir, kept); err != nil {
		writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("save peers: %v", err)})
		return
	}

	if err := s.syncAWG(); err != nil {
		writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("sync awg: %v", err)})
		return
	}

	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) freezePeers(w http.ResponseWriter, r *http.Request) {
	s.freezePeersAction(w, r, true)
}

func (s *Server) unfreezePeers(w http.ResponseWriter, r *http.Request) {
	s.freezePeersAction(w, r, false)
}

func (s *Server) freezePeersAction(w http.ResponseWriter, r *http.Request, frozen bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var req struct {
		Names []string `json:"names"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}
	if len(req.Names) == 0 {
		writeJSON(w, 400, map[string]string{"error": "names required"})
		return
	}

	matched, err := setPeersFrozen(s.confPath(), s.confDir, req.Names, frozen)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if matched == 0 {
		writeJSON(w, 404, map[string]string{"error": "peer not found"})
		return
	}

	if err := s.syncAWG(); err != nil {
		writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("sync awg: %v", err)})
		return
	}

	status := "frozen"
	if !frozen {
		status = "unfrozen"
	}
	writeJSON(w, 200, map[string]string{"status": status, "count": fmt.Sprintf("%d", matched)})
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.syncAWG(); err != nil {
		writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("sync awg: %v", err)})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "synced"})
}

func (s *Server) syncAWG() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exec := container.ExecOptions{
		Cmd:          []string{"sh", "-c", "awg syncconf awg0 <(awg-quick strip awg0)"},
		AttachStdout: true,
		AttachStderr: true,
	}
	resp, err := s.cli.ContainerExecCreate(ctx, s.containerName, exec)
	if err != nil {
		return fmt.Errorf("exec syncconf: %w", err)
	}
	if err := s.cli.ContainerExecStart(ctx, resp.ID, container.ExecStartOptions{}); err != nil {
		return fmt.Errorf("start syncconf: %w", err)
	}

	return nil
}

// execOutput runs a command in the container and returns its stdout.
func (s *Server) execOutput(ctx context.Context, cmd ...string) (string, error) {
	exec := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}
	resp, err := s.cli.ContainerExecCreate(ctx, s.containerName, exec)
	if err != nil {
		return "", fmt.Errorf("exec create: %w", err)
	}

	attach, err := s.cli.ContainerExecAttach(ctx, resp.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("exec attach: %w", err)
	}
	defer attach.Close()

	var out, errBuf bytes.Buffer
	done := make(chan error, 1)
	go func() {
		_, err := stdcopy.StdCopy(&out, &errBuf, attach.Reader)
		done <- err
	}()

	if err := s.cli.ContainerExecStart(ctx, resp.ID, container.ExecStartOptions{}); err != nil {
		return "", fmt.Errorf("exec start: %w", err)
	}
	if err := <-done; err != nil {
		return "", fmt.Errorf("exec read: %w", err)
	}

	inspect, err := s.cli.ContainerExecInspect(ctx, resp.ID)
	if err != nil {
		return "", fmt.Errorf("exec inspect: %w", err)
	}
	if inspect.ExitCode != 0 {
		return "", fmt.Errorf("command failed (exit %d): %s", inspect.ExitCode, strings.TrimSpace(errBuf.String()))
	}
	return out.String(), nil
}

func (s *Server) confPath() string {
	return strings.TrimRight(s.confDir, "/") + "/awg0.conf"
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
