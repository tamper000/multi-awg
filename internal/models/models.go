package models

import "time"

type User struct {
	ID           int64      `db:"id" goqu:"skipinsert" json:"id"`
	Username     string     `db:"username" json:"username"`
	Role         string     `db:"role" json:"role"`
	ExpiresAt    *time.Time `db:"expires_at" json:"expires_at"`
	Frozen       bool       `db:"frozen" json:"frozen"`
	SubToken     string     `db:"sub_token" json:"-"`
	PasswordHash string     `db:"password_hash" json:"-"`
	CreatedAt    time.Time  `db:"created_at" goqu:"skipinsert" json:"created_at"`
}

type Peer struct {
	ID                  int64     `db:"id" goqu:"skipinsert" json:"id"`
	UserID              int64     `db:"user_id" json:"-"`
	Name                string    `db:"name" json:"name"`
	PeerName            string    `db:"peer_name" json:"-"`
	TrafficReceived     int64     `db:"traffic_received" json:"traffic_received"`
	TrafficSent         int64     `db:"traffic_sent" json:"traffic_sent"`
	LastReceivedCounter int64     `db:"last_received_counter" json:"-"`
	LastSentCounter     int64     `db:"last_sent_counter" json:"-"`
	CreatedAt           time.Time `db:"created_at" goqu:"skipinsert" json:"created_at"`
}
