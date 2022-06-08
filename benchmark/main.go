package main

import (
	"fmt"
	"github.com/hugjobk/cotcp"
	"github.com/hugjobk/go-benchmark"
	"io"
	"os"
	"time"
)

const ServerAddr = "0.0.0.0:9001"

func initServer(addr string) {
	srv := cotcp.Server{
		Network: "tcp",
		Address: addr,
		Handler: cotcp.HandlerFunc(func(w io.Writer, m cotcp.Packet) {
			fmt.Fprintf(w, "message from %s to %s: %s", m.RemoteAddr, m.LocalAddr, m.Data)
		}),
	}
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}
}

func main() {
	go initServer(ServerAddr)
	cli := cotcp.Client{
		Network:   "tcp",
		Address:   ServerAddr,
		ConnCount: 10,
	}
	if err := cli.Init(); err != nil {
		panic(err)
	}
	deadline := time.Now().Add(1 * time.Second)
	if !cli.Ready(deadline) {
		fmt.Printf("Cannot connect to %s\n", ServerAddr)
		return
	}
	b := benchmark.Benchmark{
		WorkerCount:  500,
		Duration:     30 * time.Second,
		LatencyStart: 5 * time.Millisecond,
		LatencyStep:  6,
		ShowProcess:  true,
	}
	b.Run("Benchmark Send", func(i int) error {
		deadline := time.Now().Add(1 * time.Second)
		_, err := cli.Send(deadline, []byte("Hello"))
		return err
	}).Report(os.Stdout).PrintErrors(os.Stdout, 10)
	b.Run("Benchmark SendNoReply", func(i int) error {
		deadline := time.Now().Add(1 * time.Second)
		return cli.SendNoReply(deadline, []byte("Hello"))
	}).Report(os.Stdout).PrintErrors(os.Stdout, 10)
}
