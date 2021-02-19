package raft

import (
	"context"
	"sync"
)

const (
	LEADER = iota
	CANDIDATE
	FOLLOWER
)

func NewState() *state {
	ctx, cancel := context.WithCancel(context.Background())

	s := &state{
		wg:       sync.WaitGroup{},
		ctx:      ctx,
		cancel:   cancel,
		opChan:   make(chan stateOpReq),
		state:    FOLLOWER,
		term:     1,
		votedFor: 0,
	}

	s.run()
	return s
}

type stateOp func(*state)

type stateOpReq struct {
	op   stateOp
	done chan struct{}
}

type state struct {
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	opChan chan stateOpReq

	state    int
	term     uint64
	votedFor uint64
}

func (s *state) run() {
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		for {
			select {
			case <-s.ctx.Done():
				return
			case opReq := <-s.opChan:
				opReq.op(s)
				opReq.done <- struct{}{}
			}
		}
	}()
}

func (s *state) Do(op stateOp) {
	done := make(chan struct{})
	s.opChan <- stateOpReq{
		op:   op,
		done: done,
	}
	<-done
}

func (s *state) stop() {
	s.cancel()
	s.wg.Wait()
}
