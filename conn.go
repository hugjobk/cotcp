package cotcp

import (
	"crypto/tls"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type conn struct {
	mtx     sync.Mutex
	netConn net.Conn
}

func (c *conn) Write(deadline time.Time, b []byte) (int, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if err := c.netConn.SetWriteDeadline(deadline); err != nil {
		return 0, err
	}
	return c.netConn.Write(b)
}

type connList struct {
	mtx sync.RWMutex
	i   uint32
	a   []*conn
	m   map[*conn]int
}

func newConnList() *connList {
	return &connList{m: make(map[*conn]int)}
}

func (l *connList) Len() int {
	l.mtx.RLock()
	defer l.mtx.RUnlock()
	return len(l.a)
}

func (l *connList) Next() *conn {
	l.mtx.RLock()
	defer l.mtx.RUnlock()
	if len(l.a) == 0 {
		return nil
	}
	i := atomic.AddUint32(&l.i, 1)
	return l.a[int(i)%len(l.a)]
}

func (l *connList) Append(c *conn) bool {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if _, ok := l.m[c]; ok {
		return false
	}
	l.m[c] = len(l.a)
	l.a = append(l.a, c)
	return true
}

func (l *connList) Index(i int) *conn {
	l.mtx.RLock()
	defer l.mtx.RUnlock()
	return l.a[i]
}

func (l *connList) Remove(c *conn) bool {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	idx, ok := l.m[c]
	if !ok {
		return false
	}
	n := len(l.a) - 1
	l.a[idx] = l.a[n]
	l.m[l.a[idx]] = idx
	l.a = l.a[:n]
	delete(l.m, c)
	c.netConn.Close()
	return true
}

func (l *connList) Close() error {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	for _, c := range l.a {
		c.netConn.Close()
	}
	l.a = nil
	l.m = nil
	return nil
}

func dial(network string, addr string, timeout time.Duration, tlsConfig *tls.Config) (*conn, error) {
	var (
		netConn net.Conn
		err     error
	)
	if tlsConfig != nil {
		d := tls.Dialer{
			NetDialer: &net.Dialer{Timeout: timeout},
			Config:    tlsConfig,
		}
		netConn, err = d.Dial(network, addr)
	} else {
		d := net.Dialer{Timeout: timeout}
		netConn, err = d.Dial(network, addr)
	}
	if err != nil {
		return nil, err
	}
	return &conn{netConn: netConn}, nil
}
