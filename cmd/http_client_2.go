package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"golang.org/x/net/http2"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	maxConnections  = 10                 // 设定连接池的大小
	keepAlivePeriod = 3000 * time.Second // 设置 KeepAlive 周期
)

type ConnPool struct {
	conns []*http2.Transport
	next  int
	mu    sync.Mutex
}

func NewConnPool() *ConnPool {
	pool := &ConnPool{
		conns: make([]*http2.Transport, maxConnections),
	}

	dialer := &net.Dialer{
		KeepAlive: keepAlivePeriod,
	}

	for i := 0; i < maxConnections; i++ {
		// 使用自定义的或默认的 TLS 配置
		transport := &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				//set connection timeout as 2 seconds
				//return net.DialTimeout(network, addr, time.Second*time.Duration(2))
				// 使用自定义 Dialer 和 context
				return dialer.DialContext(ctx, network, addr)
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
			//	return tls.Dial(network, addr, cfg)
			//},
		}

		pool.conns[i] = transport
	}

	return pool
}

func (p *ConnPool) GetTransport() *http2.Transport {
	p.mu.Lock()
	defer p.mu.Unlock()
	transport := p.conns[p.next]
	p.next = (p.next + 1) % maxConnections
	return transport
}

func main() {

	request, err := http.NewRequest("GET", "http://http-server:12345", nil)
	if err != nil {
		log.Fatal(err)
	}
	pool := NewConnPool()
	time.Sleep(time.Second * 3)
	//ch := make(chan struct{}, 3000)
	for {
		//ch <- struct{}{}
		start := time.Now()
		transport := pool.GetTransport()
		client := &http.Client{Transport: transport}
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
		// 关闭连接
		//if http2Trans, ok := client.Transport.(*http2.Transport); ok {
		//	http2Trans.CloseIdleConnections()
		//}
		//<-ch
	}
}
