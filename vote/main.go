package main

import (
	"flag"
	"log"
	"net"
	"time"

	pb "github.com/KumKeeHyun/grpc-example/vote/protobuf"
	"github.com/KumKeeHyun/grpc-example/vote/raft"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
)

var (
	ID    = flag.Uint64("id", 1, "server id")
	Addr  = flag.String("addr", "localhost:10000", "server addr")
	Peers = []raft.Peer{
		raft.Peer{
			ID:   1,
			Addr: "localhost:10000",
		},
		raft.Peer{
			ID:   2,
			Addr: "localhost:10001",
		},
		raft.Peer{
			ID:   3,
			Addr: "localhost:10002",
		},
	}
)

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", *Addr)
	if err != nil {
		log.Fatal(err)
	}

	cfg := &raft.Config{
		NodeInfo: raft.Peer{
			ID:   *ID,
			Addr: *Addr,
		},
		Peers: Peers,
		Timeout: raft.Timeout{
			ElectionTimeout: 300 * time.Millisecond,
			MinElectTimeout: 15 * time.Millisecond,
		},
	}
	srv, err := raft.NewNode(cfg)
	if err != nil {
		log.Fatal(err)
	}

	grpcSrv := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_recovery.UnaryServerInterceptor(),
		),
	)
	pb.RegisterRaftServiceServer(grpcSrv, srv)

	srv.StartRaftNode()
	grpcSrv.Serve(lis)
}
