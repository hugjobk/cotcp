package cotcp_test

import (
	"fmt"
	"github.com/hugjobk/cotcp"
	"io"
	"testing"
)

const ServerAddr = "0.0.0.0:9000"

func TestServer_ListenAndServe(t *testing.T) {
	s := cotcp.Server{
		Network: "tcp",
		Address: ServerAddr,
		Handler: cotcp.HandlerFunc(func(w io.Writer, m cotcp.Packet) {
			fmt.Fprintf(w, "messsage form %s to %s: %s", m.RemoteAddr, m.LocalAddr, m.Data)
		}),
	}
	if err := s.ListenAndServe(); err != nil {
		t.Fatal(err)
	}
}
