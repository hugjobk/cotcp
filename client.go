package cotcp

import (
	"bufio"
	"crypto/tls"
	"errors"
	"log"
	"os"
	"sync"
	"time"
)

type Client struct {
	Network     string
	Address     string
	DialTimeout time.Duration
	TLSConfig   *tls.Config
	ConnCount   int
	Logger      *log.Logger

	conns *connList

	cbMtx     sync.Mutex
	callbacks map[uint32]chan []byte

	mtx    sync.Mutex
	closed bool
}

func (cli *Client) Init() error {
	if cli.Network == "" {
		return errors.New("network cannot be empty")
	}
	if cli.Address == "" {
		return errors.New("address cannot be empty")
	}
	if cli.ConnCount == 0 {
		return errors.New("conn count cannot be zero")
	}
	if cli.Logger == nil {
		cli.Logger = log.New(os.Stderr, "[CoTCP Client] ", log.LstdFlags)
	}
	cli.conns = newConnList()
	cli.callbacks = make(map[uint32]chan []byte)
	go func() {

		tempDelay := 5 * time.Millisecond
	loop:
		for {
			cli.mtx.Lock()
			if cli.closed {
				cli.mtx.Unlock()
				return
			}
			for cli.conns.Len() < cli.ConnCount {
				c, err := dial(cli.Network, cli.Address, cli.DialTimeout, cli.TLSConfig)
				if err != nil {
					cli.mtx.Unlock()
					cli.Logger.Printf("Dial error: %s; retrying in %s", err, tempDelay)
					time.Sleep(tempDelay)
					tempDelay *= 2
					if tempDelay > 1*time.Second {
						tempDelay = 1 * time.Second
					}
					continue loop
				}
				tempDelay = 5 * time.Millisecond
				cli.conns.Append(c)
				go cli.handleConn(c)
			}
			cli.mtx.Unlock()
			time.Sleep(1 * time.Second)
		}
	}()
	return nil
}

func (cli *Client) Ready(deadline time.Time) bool {
	t1 := time.NewTimer(time.Until(deadline))
	t2 := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-t1.C:
			t2.Stop()
			return false
		case <-t2.C:
			if cli.conns.Len() > 0 {
				t1.Stop()
				t2.Stop()
				return true
			}
		}
	}
}

func (cli *Client) Close() error {
	cli.mtx.Lock()
	cli.closed = true
	cli.conns.Close()
	cli.mtx.Unlock()
	return nil
}

func (cli *Client) Send(deadline time.Time, msg []byte) ([]byte, error) {
	id := allocPacketId()
	ch := make(chan []byte)
	cli.cbMtx.Lock()
	cli.callbacks[id] = ch
	cli.cbMtx.Unlock()
	if err := cli.send(deadline, id, msg); err != nil {
		cli.cbMtx.Lock()
		delete(cli.callbacks, id)
		cli.cbMtx.Unlock()
		return nil, err
	}
	t := time.NewTimer(time.Until(deadline))
	select {
	case rsp := <-ch:
		t.Stop()
		return rsp, nil
	case <-t.C:
		cli.cbMtx.Lock()
		delete(cli.callbacks, id)
		cli.cbMtx.Unlock()
		return nil, ErrDeadlineExceed
	}
}

func (cli *Client) SendNoReply(deadline time.Time, msg []byte) error {
	return cli.send(deadline, 0, msg)
}

func (cli *Client) Ping(msg []byte, maxRetry int, interval time.Duration, timeout time.Duration) {
	for retry := 0; retry < maxRetry; {
		deadline := time.Now().Add(timeout)
		if _, err := cli.Send(deadline, msg); err != nil {
			cli.Logger.Printf("Ping error: %s; retry=%d", err, retry)
			retry++
		} else {
			retry = 0
		}
		time.Sleep(interval)
	}
}

func (cli *Client) send(deadline time.Time, id uint32, msg []byte) error {
	c := cli.conns.Next()
	if c == nil {
		return ErrNoConnection
	}
	var buf []byte
	if cap(msg)-len(msg) >= 10 {
		buf = pack(id, msg)
	} else {
		buf = getBuffer()
		buf = packV2(id, buf, msg)
		defer putBuffer(buf)
	}
	_, err := c.Write(deadline, buf)
	return err
}

func (cli *Client) handleConn(c *conn) {
	localAddr := c.netConn.LocalAddr().String()
	remoteAddr := c.netConn.RemoteAddr().String()
	cli.Logger.Printf("New connection %s->%s", localAddr, remoteAddr)
	scanner := bufio.NewScanner(c.netConn)
	scanner.Split(packetSplit)
	for scanner.Scan() {
		id, data := unpack(scanner.Bytes())
		cli.cbMtx.Lock()
		cb, ok := cli.callbacks[id]
		if !ok {
			cli.cbMtx.Unlock()
			cli.Logger.Printf("Receive unknown packet: %d", id)
			continue
		}
		delete(cli.callbacks, id)
		cli.cbMtx.Unlock()
		buf := make([]byte, len(data))
		copy(buf, data)
		cb <- buf
	}
	if err := scanner.Err(); err != nil {
		cli.Logger.Print(err)
	}
	cli.conns.Remove(c)
	cli.Logger.Printf("Closed connection %s->%s", localAddr, remoteAddr)
}
