package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	url   string
	token string
}

func New(url, token string) *Client {
	return &Client{url: url, token: token}
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.url+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return http.DefaultClient.Do(req)
}

func (c *Client) CreatePeer(ctx context.Context, name string) (int, interface{}, error) {
	body, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return 0, nil, err
	}
	resp, err := c.do(ctx, http.MethodPost, "/api/peers", bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var e Error
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return resp.StatusCode, e, nil
	}
	var p Peer
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return 0, nil, fmt.Errorf("decode worker response: %w", err)
	}
	return resp.StatusCode, p, nil
}

func (c *Client) GetPeerConfig(ctx context.Context, name string) (int, interface{}, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/peers/"+url.PathEscape(name)+"/config", nil)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var e Error
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return resp.StatusCode, e, nil
	}
	var cfg PeerConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return 0, nil, fmt.Errorf("decode worker response: %w", err)
	}
	return resp.StatusCode, cfg, nil
}

func (c *Client) GetPeersSub(ctx context.Context, peers []SubPeer) (int, interface{}, error) {
	body, err := json.Marshal(map[string][]SubPeer{"peers": peers})
	if err != nil {
		return 0, nil, err
	}
	resp, err := c.do(ctx, http.MethodPost, "/api/peers/sub", bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var e Error
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return resp.StatusCode, e, nil
	}
	var sub Sub
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return 0, nil, fmt.Errorf("decode worker response: %w", err)
	}
	return resp.StatusCode, sub, nil
}

func (c *Client) GetPeerStats(ctx context.Context, name string) (int, interface{}, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/peers/"+url.PathEscape(name)+"/stats", nil)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var e Error
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return resp.StatusCode, e, nil
	}
	var st Stats
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		return 0, nil, fmt.Errorf("decode worker response: %w", err)
	}
	return resp.StatusCode, st, nil
}

func (c *Client) GetStats(ctx context.Context) (int, interface{}, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/stats", nil)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var e Error
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return resp.StatusCode, e, nil
	}
	var stats []Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return 0, nil, fmt.Errorf("decode worker response: %w", err)
	}
	return resp.StatusCode, stats, nil
}

func (c *Client) Sync(ctx context.Context) (int, interface{}, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/sync", nil)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var e Error
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return resp.StatusCode, e, nil
	}
	var status struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return 0, nil, fmt.Errorf("decode worker response: %w", err)
	}
	return resp.StatusCode, status, nil
}

func (c *Client) DeletePeer(ctx context.Context, name string) (int, interface{}, error) {
	return c.deletePeers(ctx, []string{name})
}

func (c *Client) DeletePeers(ctx context.Context, names []string) (int, interface{}, error) {
	return c.deletePeers(ctx, names)
}

func (c *Client) deletePeers(ctx context.Context, names []string) (int, interface{}, error) {
	body, err := json.Marshal(map[string][]string{"names": names})
	if err != nil {
		return 0, nil, err
	}
	resp, err := c.do(ctx, http.MethodDelete, "/api/peers", bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var e Error
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return resp.StatusCode, e, nil
	}
	var status struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return 0, nil, fmt.Errorf("decode worker response: %w", err)
	}
	return resp.StatusCode, status, nil
}

func (c *Client) FreezePeers(ctx context.Context, names []string) (int, interface{}, error) {
	body, err := json.Marshal(map[string][]string{"names": names})
	if err != nil {
		return 0, nil, err
	}
	resp, err := c.do(ctx, http.MethodPost, "/api/peers/freeze", bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var e Error
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return resp.StatusCode, e, nil
	}

	var response struct {
		Status string `json:"status"`
		Count  string `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, nil, fmt.Errorf("decode worker response: %w", err)
	}
	return resp.StatusCode, response, nil
}

func (c *Client) UnfreezePeers(ctx context.Context, names []string) (int, interface{}, error) {
	body, err := json.Marshal(map[string][]string{"names": names})
	if err != nil {
		return 0, nil, err
	}
	resp, err := c.do(ctx, http.MethodPost, "/api/peers/unfreeze", bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var e Error
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return resp.StatusCode, e, nil
	}

	var response struct {
		Status string `json:"status"`
		Count  string `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, nil, fmt.Errorf("decode worker response: %w", err)
	}
	return resp.StatusCode, response, nil
}
