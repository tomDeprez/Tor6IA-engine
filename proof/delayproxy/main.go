// delayproxy relays one TCP port to another and adds a fixed delay in each direction,
// like a link between two machines on the Internet. It counts the bytes it carries.
// It measures what splitting a model across real machines would cost.
//
//	delayproxy -listen 127.0.0.1:60052 -to 127.0.0.1:50052 -delay 25ms
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"
)

var up, down atomic.Int64 // bytes towards the server, bytes back

type chunk struct {
	data []byte
	due  time.Time
}

// pipe copies src to dst. Each chunk leaves delay after it arrived, without holding back
// the chunks behind it: latency grows, throughput does not.
func pipe(dst, src net.Conn, delay time.Duration, count *atomic.Int64) {
	queue := make(chan chunk, 4096)
	go func() {
		defer close(queue)
		for {
			buf := make([]byte, 64<<10)
			n, err := src.Read(buf)
			if n > 0 {
				queue <- chunk{buf[:n], time.Now().Add(delay)}
			}
			if err != nil {
				return
			}
		}
	}()
	for c := range queue {
		time.Sleep(time.Until(c.due))
		if _, err := dst.Write(c.data); err != nil {
			break
		}
		count.Add(int64(len(c.data)))
	}
	dst.Close()
	src.Close()
}

func main() {
	listen := flag.String("listen", "127.0.0.1:60052", "address to listen on")
	to := flag.String("to", "127.0.0.1:50052", "address to relay to")
	delay := flag.Duration("delay", 0, "delay added in each direction")
	flag.Parse()

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		for range time.Tick(time.Second) {
			fmt.Printf("%d %d %d\n", time.Now().UnixMilli(), up.Load(), down.Load())
		}
	}()
	for {
		client, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		server, err := net.Dial("tcp", *to)
		if err != nil {
			client.Close()
			continue
		}
		for _, c := range []net.Conn{client, server} {
			c.(*net.TCPConn).SetNoDelay(true)
		}
		go pipe(server, client, *delay, &up)
		go pipe(client, server, *delay, &down)
	}
}
