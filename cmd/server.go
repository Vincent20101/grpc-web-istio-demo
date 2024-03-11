// Copyright 2019 VMware, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/shimingyah/pool"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"log"
	"net"
	"runtime"

	"github.com/venilnoronha/grpc-web-istio-demo/proto"
	_ "go.uber.org/automaxprocs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	emoji "gopkg.in/kyokomi/emoji.v1"
)

var certFile = flag.String("cert", "/etc/secrets/certs/tls.crt", "public key ")
var keyFile = flag.String("key", "/etc/secrets/certs/tls.key", "private key")
var serverTLS = flag.Bool("tls", false, "grpc with tls certificate")

type server struct{}

func (s *server) InsertEmojis(ctx context.Context, req *proto.EmojiRequest) (*proto.EmojiResponse, error) {
	log.Printf("Client says: %s", req.InputText)
	outputText := emoji.Sprint(req.InputText)
	log.Printf("Response: %s", outputText)
	return &proto.EmojiResponse{OutputText: outputText}, nil
}

func init() {
	flag.Parse()
}

func main() {
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	log.Printf("Listening on %s\n", lis.Addr())
	log.Printf("lhb runtime.GOMAXPROCS(0): %v\n", runtime.GOMAXPROCS(-1))
	log.Printf("runtime.GOMAXPROCS(0): %v\n", runtime.GOMAXPROCS(0))
	log.Printf("runtime.NumCPU(): %v\n", runtime.NumCPU())
	log.Printf("runtime.Version(): %v\n", runtime.Version())

	opts := []grpc.ServerOption{
		grpc.InitialWindowSize(pool.InitialWindowSize),
		grpc.InitialConnWindowSize(pool.InitialConnWindowSize),
		grpc.MaxSendMsgSize(pool.MaxSendMsgSize),
		grpc.MaxRecvMsgSize(pool.MaxRecvMsgSize),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    pool.KeepAliveTime,
			Timeout: pool.KeepAliveTimeout,
		}),
	}
	if *serverTLS {
		fmt.Println("start grpc server with tls configuration")
		tls, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if nil != err {
			fmt.Printf("failed to create TLS: %v\n", err)
		}
		opts = append(opts, grpc.Creds(tls))
	}
	s := grpc.NewServer(opts...)
	proto.RegisterEmojiServiceServer(s, &server{})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
