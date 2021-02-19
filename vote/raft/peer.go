package raft

import (
	pb "github.com/KumKeeHyun/grpc-example/vote/protobuf"
	"google.golang.org/grpc"
)

type peer struct {
	id     uint64
	addr   string
	client pb.RaftServiceClient
}

func NewPeer(id uint64, addr string) (*peer, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return &peer{
		id:     id,
		addr:   addr,
		client: pb.NewRaftServiceClient(conn),
	}, nil
}
