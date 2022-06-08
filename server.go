package cotcp

import (
	"bufio"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

type Handler interface {
	Serve(w io.Writer, m Packet)
}

type HandlerFunc func(w io.Writer, m Packet)

func (f HandlerFunc) Serve(w io.Writer, m Packet) {
	f(w, m)
}

type Server struct {
	Network   string
	Address   string
	TLSConfig *tls.Config
	Handler   Handler
	Logger    *log.Logger

	l     net.Listener
	mtx   sync.Mutex
	conns map[net.Conn]struct{}
}

func (srv *Server) ListenAndServe() error {
	if err := srv.init(); err != nil {
		return err
	}
	if err := srv.listen(); err != nil {
		return err
	}
	tempDelay := 5 * time.Millisecond
	for {
		c, err := srv.l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				srv.Logger.Printf("Accept error: %s; retrying in %s", err, tempDelay)
				time.Sleep(tempDelay)
				tempDelay *= 2
				if tempDelay > 1*time.Second {
					tempDelay = 1 * time.Second
				}
				continue
			}
			return err
		}
		tempDelay = 5 * time.Millisecond
		srv.addConn(c)
		go srv.handleConn(c)
	}
}

func (srv *Server) init() error {
	if srv.Network == "" {
		return errors.New("network cannot be empty")
	}
	if srv.Address == "" {
		return errors.New("address cannot be empty")
	}
	if srv.Handler == nil {
		return errors.New("handler cannot be nil")
	}
	if srv.Logger == nil {
		srv.Logger = log.New(os.Stderr, "[CoTCP Server] ", log.LstdFlags)
	}
	srv.conns = make(map[net.Conn]struct{})
	return nil
}

func (srv *Server) listen() error {
	var err error
	if srv.TLSConfig != nil {
		srv.l, err = tls.Listen(srv.Network, srv.Address, srv.TLSConfig)
	} else {
		srv.l, err = net.Listen(srv.Network, srv.Address)
	}
	return err
}

func (srv *Server) addConn(c net.Conn) {
	srv.mtx.Lock()
	srv.conns[c] = struct{}{}
	srv.mtx.Unlock()
}

func (srv *Server) removeConn(c net.Conn) {
	srv.mtx.Lock()
	delete(srv.conns, c)
	srv.mtx.Unlock()
}

func (srv *Server) handleConn(c net.Conn) {
	localAddr := c.LocalAddr().String()
	remoteAddr := c.RemoteAddr().String()
	srv.Logger.Printf("New connection %s->%s", remoteAddr, localAddr)
	scanner := bufio.NewScanner(c)
	scanner.Split(packetSplit)
	for scanner.Scan() {
		id, data := unpack(scanner.Bytes())
		buf := make([]byte, len(data))
		copy(buf, data)
		go srv.Handler.Serve(respWritter{id: id, w: c}, Packet{Data: buf, LocalAddr: localAddr, RemoteAddr: remoteAddr})
	}
	if err := scanner.Err(); err != nil {
		srv.Logger.Print(err)
	}
	srv.removeConn(c)
	srv.Logger.Printf("Closed connection %s->%s", remoteAddr, localAddr)
}

type respWritter struct {
	id uint32
	w  io.Writer
}

func (w respWritter) Write(p []byte) (int, error) {
	if w.id == 0 {
		return 0, ErrNoReply
	}
	var buf []byte
	if cap(p)-len(p) >= 10 {
		buf = pack(w.id, p)
	} else {
		buf = getBuffer()
		buf = packV2(w.id, buf, p)
		defer putBuffer(buf)
	}
	return w.w.Write(buf)
}
