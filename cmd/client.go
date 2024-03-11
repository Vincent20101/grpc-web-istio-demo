// Copyright 2019 VMware, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"log"
	"math"
	"time"

	"github.com/shimingyah/pool"
	"github.com/venilnoronha/grpc-web-istio-demo/proto"
	"golang.org/x/net/context"
)

var svc = flag.String("server", "172.29.13.181:50051", "The server's address")
var text = flag.String("text", "Hello world!", "The input text")
var clientTLS = flag.Bool("tls", false, "grpc with tls certificate")

func init() {
	flag.Parse()
}

func main() {
	//conn, err := grpc.Dial(*svc, grpc.WithInsecure(), grpc.WithBlock())
	//if err != nil {
	//	log.Fatalf("Couldn't connect to the service: %v", err)
	//}
	//defer conn.Close()

	//c := proto.NewEmojiServiceClient(conn)
	var opts []grpc.DialOption
	opts = append(opts,
		grpc.WithBackoffMaxDelay(pool.BackoffMaxDelay),
		grpc.WithInitialWindowSize(pool.InitialWindowSize),
		grpc.WithInitialConnWindowSize(pool.InitialConnWindowSize),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(pool.MaxSendMsgSize)),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(pool.MaxRecvMsgSize)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                pool.KeepAliveTime,
			Timeout:             pool.KeepAliveTimeout,
			PermitWithoutStream: false,
		}))
	if *clientTLS {
		fmt.Println("start grpc client pool with tls")
		creds, err := credentials.NewClientTLSFromFile("/etc/secrets/certs/ca.crt", "")
		if err != nil {
			fmt.Printf("failed to load client TLS credentials for anpd: %v\n", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		//opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		opts = append(opts, grpc.WithInsecure())
	}

	p, err := pool.New(*svc, pool.Options{
		Dial: func(address string) (*grpc.ClientConn, error) {
			ctx, cancel := context.WithTimeout(context.Background(), pool.DialTimeout)
			defer cancel()
			return grpc.DialContext(ctx, address, opts...)
		},
		MaxIdle:              5,
		MaxActive:            5000,
		MaxConcurrentStreams: math.MaxUint32,
		Reuse:                true,
	})
	//p, err := pool.New(*svc, pool.DefaultOptions)
	if err != nil {
		log.Fatalf("failed to new pool: %v", err)
	}
	defer p.Close()

	time.Sleep(time.Second * 10)
	ch := make(chan struct{}, 100)
	timer := time.NewTimer(time.Second * 100)
	conn, err := p.Get()
	log.Println("conn: ", conn.Value().GetState())
	log.Println("err: ", err)
	for {
		ch <- struct{}{}
		go func() {
			ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

			//log.Printf("Request: %s", *text)

			conn, err := p.Get()
			defer conn.Close()
			if err != nil {
				log.Printf("get conn from pool: %v\n", err)
				return
			}
			c := proto.NewEmojiServiceClient(conn.Value())
			start := time.Now()
			res, err := c.InsertEmojis(ctx, &proto.EmojiRequest{
				InputText: *text,
			})
			if err != nil {
				log.Fatalf("Couldn't call service: %v", err)
			}
			log.Printf("Server says: %s, use time: %v", res.OutputText, time.Now().Sub(start))

			//time.Sleep(time.Second * 1)
			<-ch
		}()
		select {
		case <-timer.C:
			time.Sleep(time.Hour)
		default:
			continue
		}

	}
}
