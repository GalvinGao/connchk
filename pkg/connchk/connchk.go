package connchk

import (
	"net"
	"strings"
	"time"
)

type ConnChk struct {
	c    int
	addr string
}

// New creates a new TCPConnChk.
func New(addr string) *ConnChk {
	return &ConnChk{addr: addr}
}

// Do implements ConnChk.Do.
func (c *ConnChk) Do() error {
	parts := strings.SplitN(c.addr, "://", 2)
	if len(parts) != 2 {
		panic("invalid addr")
	}

	network := parts[0]
	addr := parts[1]

	conn, err := net.DialTimeout(network, addr, 2*time.Second)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
