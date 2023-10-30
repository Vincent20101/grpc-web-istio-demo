package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/http2"
)

func main() {
	//h2Transport, err := http2.ConfigureTransports(&http.Transport{
	//	IdleConnTimeout: time.Minute,
	//})
	client := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(_ context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				//set connection timeout as 2 seconds
				return net.DialTimeout(network, addr, time.Second*time.Duration(2))
			},
			//DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			//	// 使用自定义 Dialer
			//	conn, err := tls.DialWithDialer(dialer, network, addr, cfg)
			//	if err != nil {
			//		return nil, err
			//	}
			//	return conn, nil
			//},
			//DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			//	return net.Dial(network, addr)
			//},
		},
	}

	request, err := http.NewRequest("GET", "http://http-server:12345", nil)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second * 3)
	fmt.Println("lhb local ip:", getLAddrIP())
	ch := make(chan struct{}, 1000)
	for {
		ch <- struct{}{}
		start := time.Now()
		response, err := client.Do(request)
		if err != nil {
			log.Fatal(err)
		}
		defer response.Body.Close()

		fmt.Println("Response Status:", response.Status, "sub time:", time.Now().Sub(start))

		// 读取响应内容
		buf := make([]byte, 1024)
		n, err := response.Body.Read(buf)
		if err != nil {
			fmt.Println("lhb")
			log.Fatal(err)
		}

		fmt.Println("Response Body:", string(buf[:n]))
		<-ch
	}
}

func getLAddrIP() net.IP {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	addrs, err := net.LookupHost(hostname)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	if len(addrs) > 0 {
		return net.ParseIP(addrs[0])
	}

	return nil
}
