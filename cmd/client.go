// Copyright 2019 VMware, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"flag"
	"log"
	"time"

	"github.com/venilnoronha/grpc-web-istio-demo/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var svc = flag.String("server", "server:9000", "The server's address")
var text = flag.String("text", "Hello world!", "The input text")

func init() {
	flag.Parse()
}

func main() {
	ch := make(chan struct{}, 2000)
	for {
		conn, err := grpc.Dial(*svc, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Fatalf("Couldn't connect to the service: %v", err)
		}
		defer conn.Close()

		c := proto.NewEmojiServiceClient(conn)

		ch <- struct{}{}
		go func() {
			ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

			log.Printf("Request: %s", *text)

			start := time.Now()
			res, err := c.InsertEmojis(ctx, &proto.EmojiRequest{
				InputText: *text,
			})
			if err != nil {
				log.Fatalf("Couldn't call service: %v", err)
			}
			log.Printf("Server says: %s, use time: %v", res.OutputText, time.Now().Sub(start))

			time.Sleep(time.Second * 1)
			<-ch
		}()

	}

}
