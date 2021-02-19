package raft

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	pb "github.com/KumKeeHyun/grpc-example/vote/protobuf"
	"google.golang.org/grpc"
)

type Node struct {
	pb.UnimplementedRaftServiceServer

	ctx       context.Context
	ctxCancel context.CancelFunc
	timeout   *timeout

	nodeState *state
	nodeInfo  Peer
	peers     map[uint64]*peer
}

func NewNode(cfg *Config) (*Node, error) {
	peers := make(map[uint64]*peer, len(cfg.Peers))
	for _, pInfo := range cfg.Peers {
		if pInfo.ID == cfg.NodeInfo.ID {
			continue
		}
		peer, err := NewPeer(pInfo.ID, pInfo.Addr)
		if err != nil {
			return nil, err
		}
		peers[pInfo.ID] = peer
	}

	timeout := NewTimeout(cfg.Timeout.ElectionTimeout, cfg.Timeout.MinElectTimeout)

	return &Node{
		ctx:       nil,
		ctxCancel: func() {},
		timeout:   timeout,
		nodeState: NewState(),
		nodeInfo:  cfg.NodeInfo,
		peers:     peers,
	}, nil
}

func (n *Node) StartRaftNode() {
	go func() {
		<-time.After(5 * time.Second)
		timeoutTicker := time.NewTicker(n.timeout.electionTimeout)

		for {
			select {
			case <-timeoutTicker.C:
				log.Println("Election timeout!")
				n.startElection()
			case req := <-n.timeout.resetReq:
				timeoutTicker.Stop()
				timeoutTicker = time.NewTicker(n.timeout.electionTimeout)
				req <- struct{}{}
			case req := <-n.timeout.stopReq:
				timeoutTicker.Stop()
				req <- struct{}{}
			}
		}
	}()
}

func (n *Node) startElection() {
	var (
		currentTerm uint64 = 0
		votes       int64  = 1 // itselft
	)

	// n.ctxCancel() // cancel context if other context is running
	ctx, ctxCancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	n.ctx, n.ctxCancel = ctx, ctxCancel

	n.nodeState.Do(func(s *state) {
		s.state = CANDIDATE
		s.term++
		currentTerm = s.term
		s.votedFor = n.nodeInfo.ID
	})

	log.Printf("StartElection: Start Election with Term:%d\n", currentTerm)

	for _, p := range n.peers {
		go func(p *peer) {
			resp, err := p.client.RequestVote(ctx, &pb.RequestVoteReq{
				Term:      currentTerm,
				Candidate: n.nodeInfo.ID,
			}, grpc.WaitForReady(true))
			if err != nil {
				log.Printf("StartElection: RequestVote to %d fail\n", p.id)
				return
			}

			isCandidate := true
			n.nodeState.Do(func(s *state) { isCandidate = (s.state == CANDIDATE) })
			if !isCandidate {
				log.Println("StartElection: Stop request vote goroutine. Node's state changed")
				return
			}

			respTerm := resp.GetTerm()
			if currentTerm < respTerm {
				log.Printf("StartElection: Stop Election %d. There is other leader that term is %d\n", currentTerm, respTerm)
				n.becomeFollower(respTerm)
				return
			} else if currentTerm == respTerm && resp.GetVoteGranted() {
				v := atomic.AddInt64(&votes, 1)
				if n.isQuorum(v) {
					log.Println("StartElection: Win election. Start leader")
					n.startLeader()
					return
				}
			}
		}(p)
	}
}

func (n *Node) becomeFollower(leaderTerm uint64) {
	n.timeout.stop()
	defer n.timeout.reset()
	// n.ctxCancel()

	n.nodeState.Do(func(s *state) {
		s.state = FOLLOWER
		s.term = leaderTerm
		s.votedFor = 0
	})
}

func (n *Node) startLeader() {
	n.timeout.stop()
	// n.ctxCancel()
	ctx, ctxCancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	n.ctx, n.ctxCancel = ctx, ctxCancel

	n.nodeState.Do(func(s *state) {
		s.state = LEADER
	})

	go func() {
		heartbeatTicker := time.NewTicker(50 * time.Millisecond)
		defer heartbeatTicker.Stop()

		for {
			select {
			// case <-ctx.Done():
			// 	log.Println("StartLeader: Receive stop leader signal by cancel context")
			// 	return
			case <-heartbeatTicker.C:
				isLeader := true
				n.nodeState.Do(func(s *state) { isLeader = (s.state == LEADER) })
				if !isLeader {
					log.Println("StartLeader: Stop leader. Node's state is not leader")
					return
				}
				log.Println("StartLeader: Send heartbeat")
				n.sendHeartbeat(ctx)
			}
		}
	}()
}

func (n *Node) sendHeartbeat(ctx context.Context) {
	var (
		currentTerm uint64 = 0
	)
	n.nodeState.Do(func(s *state) { currentTerm = s.term })

	for _, p := range n.peers {
		go func(p *peer) {
			resp, err := p.client.AppendEntries(ctx, &pb.AppendEntriesReq{
				Term:   currentTerm,
				Leader: 0,
			}, grpc.WaitForReady(true))
			if err != nil {
				log.Printf("SendHeartbeat: AppendEntries to %d fail\n", p.id)
				return
			}

			if resp.GetTerm() > currentTerm {
				log.Printf("SendHeartbeat: Stop leader %d. There is other leader that term is %d\n", currentTerm, resp.GetTerm())
				n.becomeFollower(resp.GetTerm())
				return
			}
		}(p)
	}
}

func (n *Node) isQuorum(i int64) bool {
	return i*2 > int64(len(n.peers))+1
}

func (n *Node) RequestVote(ctx context.Context, in *pb.RequestVoteReq) (*pb.RequestVoteResp, error) {
	resp := &pb.RequestVoteResp{
		Term:        0,
		VoteGranted: false,
	}

	n.nodeState.Do(func(s *state) { resp.Term = s.term })
	reqTerm := in.GetTerm()
	if reqTerm > resp.Term {
		log.Printf("RequestVote: There is other leader that term is %d\n", reqTerm)
		n.becomeFollower(reqTerm)
		resp.Term = reqTerm
	}

	n.nodeState.Do(func(s *state) {
		if reqTerm == s.term && (s.votedFor == 0 || s.votedFor == in.GetCandidate()) {
			resp.VoteGranted = true
			s.votedFor = in.GetCandidate()
		}
	})
	if resp.VoteGranted {
		log.Printf("RequestVote: vote for %d in term %d\n", in.GetCandidate(), reqTerm)
		n.timeout.reset()
	}

	return resp, nil
}

func (n *Node) AppendEntries(ctx context.Context, in *pb.AppendEntriesReq) (*pb.AppendEntriesResp, error) {
	resp := &pb.AppendEntriesResp{
		Term:    0,
		Success: false,
	}

	log.Printf("AppendEntries: Receive AppendEntries Request in term\n", in.GetTerm())

	n.nodeState.Do(func(s *state) {
		resp.Term = s.term
	})
	reqTerm := in.GetTerm()
	if reqTerm > resp.Term {
		log.Printf("AppendEntries: There is other leader that term is %d\n", reqTerm)
		n.becomeFollower(reqTerm)
		resp.Term = reqTerm
	}

	if resp.Term == reqTerm {
		isFollower := true
		n.nodeState.Do(func(s *state) { isFollower = (s.state == FOLLOWER) })
		if !isFollower {
			n.becomeFollower(reqTerm)
		}
		n.timeout.reset()
		resp.Success = true
	}

	return resp, nil
}
