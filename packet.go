package cotcp

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

const (
	bufferSize     = 1024
	protocolNumber = 0x123456
)

type Packet struct {
	Data       []byte
	LocalAddr  string
	RemoteAddr string
}

var packetId uint32

func allocPacketId() uint32 {
	for {
		if id := atomic.AddUint32(&packetId, 1); id != 0 {
			return id
		}
	}
}

func pack(id uint32, p []byte) []byte {
	b := append(p, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	copy(b[10:], b)
	binary.BigEndian.PutUint32(b, protocolNumber)
	binary.BigEndian.PutUint32(b[4:], id)
	binary.BigEndian.PutUint16(b[8:], uint16(len(p)))
	return b
}

func packV2(id uint32, dst []byte, src []byte) []byte {
	binary.BigEndian.PutUint32(dst, protocolNumber)
	binary.BigEndian.PutUint32(dst[4:], id)
	binary.BigEndian.PutUint16(dst[8:], uint16(len(src)))
	dst = append(dst, src...)
	return dst
}

func unpack(p []byte) (uint32, []byte) {
	return binary.BigEndian.Uint32(p[4:]), p[10:]
}

func packetSplit(data []byte, atEOF bool) (int, []byte, error) {
	if len(data) > 10 {
		if n := binary.BigEndian.Uint32(data); n != protocolNumber {
			return 0, nil, fmt.Errorf("invalid protocol number: 0x%08x", n)
		}
		l := binary.BigEndian.Uint16(data[8:])
		pl := int(l) + 10
		if pl <= len(data) {
			return pl, data[:pl], nil
		}
	}
	if atEOF && len(data) > 0 {
		return 0, nil, io.ErrUnexpectedEOF
	}
	return 0, nil, nil
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 10, bufferSize)
	},
}

func getBuffer() []byte {
	return bufferPool.Get().([]byte)
}

func putBuffer(b []byte) {
	b = b[:10]
	bufferPool.Put(b)
}
