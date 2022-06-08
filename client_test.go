package cotcp_test

import (
	"github.com/hugjobk/cotcp"
	"testing"
	"time"
)

func TestClient_Send(t *testing.T) {
	cli := cotcp.Client{
		Network:   "tcp",
		Address:   ServerAddr,
		ConnCount: 10,
	}
	if err := cli.Init(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(1 * time.Second)
	if !cli.Ready(deadline) {
		t.Fatalf("cannot connect to %s", ServerAddr)
	}
	for {
		deadline := time.Now().Add(1 * time.Second)
		if resp, err := cli.Send(deadline, []byte("Hello")); err != nil {
			t.Error(err)
		} else {
			t.Log(string(resp))
		}
		time.Sleep(1 * time.Second)
	}
}

func TestClient_SendNoReply(t *testing.T) {
	cli := cotcp.Client{
		Network:   "tcp",
		Address:   ServerAddr,
		ConnCount: 10,
	}
	if err := cli.Init(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(1 * time.Second)
	if !cli.Ready(deadline) {
		t.Fatalf("cannot connect to %s", ServerAddr)
	}
	for {
		deadline := time.Now().Add(1 * time.Second)
		if err := cli.SendNoReply(deadline, []byte("Hello")); err != nil {
			t.Error(err)
		} else {
			t.Log("OK")
		}
		time.Sleep(1 * time.Second)
	}
}

func TestClient_Ping(t *testing.T) {
	cli := cotcp.Client{
		Network:   "tcp",
		Address:   ServerAddr,
		ConnCount: 10,
	}
	if err := cli.Init(); err != nil {
		t.Fatal(err)
	}
	cli.Ping([]byte("Ping"), 10, 1*time.Second, 1*time.Second)
	cli.Close()
	t.Log("Disconnected from server")
}
