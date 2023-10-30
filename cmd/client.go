// Copyright 2019 VMware, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"flag"
	"log"
	"time"

	"github.com/shimingyah/pool"
	"github.com/venilnoronha/grpc-web-istio-demo/proto"
	"golang.org/x/net/context"
)

var svc = flag.String("server", "server:9000", "The server's address")
var text = flag.String("text", "Hello world!", "The input text")

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

	//p, err := pool.New(*svc, pool.Options{
	//	Dial:                 pool.Dial,
	//	MaxIdle:              10,
	//	MaxActive:            100,
	//	MaxConcurrentStreams: 2000,
	//	Reuse:                false,
	//})
	p, err := pool.New(*svc, pool.DefaultOptions)
	if err != nil {
		log.Fatalf("failed to new pool: %v", err)
	}
	defer p.Close()

	time.Sleep(time.Second * 10)
	ch := make(chan struct{}, 2000)
	for {
		ch <- struct{}{}
		go func() {
			ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

			log.Printf("Request: %s", *text)

			conn, err := p.Get()
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

	}

}
