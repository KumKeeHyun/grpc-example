package raft

import (
	"math/rand"
	"time"
)

func newTimeout(electTimeout, minElectTimeout time.Duration) *timeout {
	return &timeout{
		electionTimeout: electTimeout,
		minElectTimeout: minElectTimeout,
		resetReq:        make(chan chan struct{}),
		stopReq:         make(chan chan struct{}),
	}
}

type timeout struct {
	electionTimeout time.Duration
	minElectTimeout time.Duration
	resetReq        chan chan struct{}
	stopReq         chan chan struct{}
}

// electionTimeout + random period
func (t *timeout) newTicker() *time.Ticker {
	randTimeout := time.Duration(rand.Intn(50)) * time.Millisecond
	return time.NewTicker(t.electionTimeout + randTimeout)
}

func (t *timeout) reset() {
	req := make(chan struct{})
	t.resetReq <- req
	<-req
}

func (t *timeout) stop() {
	req := make(chan struct{})
	t.stopReq <- req
	<-req
}
