package raft

import "time"

type timeout struct {
	electionTimeout time.Duration
	minElectTimeout time.Duration
	resetReq        chan chan struct{}
	stopReq         chan chan struct{}
}

func NewTimeout(electTimeout, minElectTimeout time.Duration) *timeout {
	return &timeout{
		electionTimeout: electTimeout,
		minElectTimeout: minElectTimeout,
		resetReq:        make(chan chan struct{}),
		stopReq:         make(chan chan struct{}),
	}
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
