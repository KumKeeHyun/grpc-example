package main

import (
	"context"
	"flag"
	"log"
	"net"
	"sync"
	"time"

	"github.com/KumKeeHyun/grpc-example/heartbeat/yaho"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
)

var (
	hostID   = flag.Int("id", 1, "server id")
	host     = flag.String("host", "localhost:10000", "The server address in the format of host:port")
	discover = flag.String("discover", "nil", "The server address in the format of host:port")
)

func main() {
	flag.Parse()
	log.Printf("start heartbeat hostID:%d    host:%s    discover:%s", *hostID, *host, *discover)

	lis, err := net.Listen("tcp", *host)
	if err != nil {
		log.Fatal(err)
	}

	srv := &heartBeatServer{
		id:    int32(*hostID),
		addr:  *host,
		peers: map[int32]*peer{},
		mu:    &sync.Mutex{},
	}
	// logrus.ErrorKey = "grpc.error"
	// logrusEntry := logrus.NewEntry(logrus.StandardLogger())
	grpcSrv := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			// grpc_logrus.UnaryServerInterceptor(logrusEntry),
			grpc_recovery.UnaryServerInterceptor(),
		),
	)
	yaho.RegisterHeartBeatServiceServer(grpcSrv, srv)

	srv.StartMain()
	grpcSrv.Serve(lis)
}

func getClientFac(hostAddr string) func() yaho.HeartBeatServiceClient {
	var (
		once sync.Once
		cli  yaho.HeartBeatServiceClient
	)

	return func() yaho.HeartBeatServiceClient {
		once.Do(func() {
			conn, _ := grpc.Dial(hostAddr, grpc.WithInsecure())
			cli = yaho.NewHeartBeatServiceClient(conn)
		})

		return cli
	}
}

type peer struct {
	getClient func() yaho.HeartBeatServiceClient
	addr      string
}

type heartBeatServer struct {
	yaho.UnimplementedHeartBeatServiceServer

	id    int32
	addr  string
	peers map[int32]*peer
	mu    *sync.Mutex
}

func (hbs *heartBeatServer) Claim(ctx context.Context, in *yaho.ClaimArg) (*yaho.ClaimReply, error) {
	claimID := in.GetClaimID()
	claimAddr := in.GetClaimAddr()
	log.Printf("claim from %d\n", claimID)

	if p, exist := hbs.peers[claimID]; !exist {
		hbs.mu.Lock()
		log.Printf("join peer id:%d    addr:%s\n", claimID, claimAddr)

		hbs.peers[claimID] = &peer{
			getClient: getClientFac(claimAddr),
			addr:      claimAddr,
		}
		hbs.mu.Unlock()
	} else {
		if p.addr != claimAddr {
			log.Printf("conflect id %d\n", claimID)
			return &yaho.ClaimReply{Success: false}, nil
		}
	}

	return &yaho.ClaimReply{Success: true}, nil
}

func (hbs *heartBeatServer) GetPeers(ctx context.Context, in *yaho.GetPeersArg) (*yaho.GetPeersReply, error) {
	hbs.mu.Lock()
	defer hbs.mu.Unlock()

	peerList := []*yaho.Peer{}
	for k, v := range hbs.peers {
		peerList = append(peerList, &yaho.Peer{Id: k, Addr: v.addr})
	}

	return &yaho.GetPeersReply{Peers: peerList}, nil
}

func (hbs *heartBeatServer) StartMain() {
	if *discover != "nil" {
		hbs.discoverPeers()
	}
	hbs.peers[int32(*hostID)] = &peer{
		getClient: nil,
		addr:      *host,
	}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			<-ticker.C
			hbs.sendHeartbeats()
		}
	}()

}

func (hbs *heartBeatServer) discoverPeers() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	reply, err := getClientFac(*discover)().GetPeers(ctx, &yaho.GetPeersArg{})
	if err != nil {
		log.Fatalln("cannot discover peers")
	}

	for _, p := range reply.GetPeers() {
		log.Printf("init peer id:%d    addr:%s\n", p.Id, p.Addr)
		hbs.peers[p.Id] = &peer{
			getClient: getClientFac(p.Addr),
			addr:      p.Addr,
		}
	}
}

func (hbs *heartBeatServer) sendHeartbeats() {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)

	for peerID, peerInfo := range hbs.peers {
		if peerID == int32(*hostID) {
			continue
		}
		go func(pid int32, pinfo *peer) {
			result, err := pinfo.getClient().Claim(ctx, &yaho.ClaimArg{
				ClaimID:   hbs.id,
				ClaimAddr: hbs.addr,
			})

			if err != nil {
				if result.Success {
					log.Printf("claim success to %d\n", pid)
				} else {
					log.Fatalln("claim fail")
				}
			}
		}(peerID, peerInfo)
	}
}
