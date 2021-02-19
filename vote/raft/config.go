package raft

import "time"

type Config struct {
	NodeInfo Peer
	Peers    []Peer
	Timeout  Timeout
}

type Peer struct {
	ID   uint64
	Addr string
}

type Timeout struct {
	ElectionTimeout time.Duration
	MinElectTimeout time.Duration
}
