package main

import "time"

const (
	subnet      = "10.0.0."
	subnetMask  = "/32"
	startIP     = 3 // first client IP: 10.0.0.3
	wgInterface = "awg0"
	dnsDefault  = "94.140.14.14"
	mtuDefault  = 1280
)

type Peer struct {
	Name       string    `json:"name"`
	PrivateKey string    `json:"private_key"`
	PublicKey  string    `json:"public_key"`
	IP         string    `json:"ip"`
	DNS        string    `json:"dns"`
	CreatedAt  time.Time `json:"created_at"`
}
