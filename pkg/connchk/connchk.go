package connchk

import (
	"errors"
	"time"
)

type ConnChk struct {
	expectInterval time.Duration
	gracePeriod    time.Duration
	lastPing       time.Time
}

// New creates a new ConnChk.
func New(expectInterval time.Duration, gracePeriod time.Duration) *ConnChk {
	return &ConnChk{
		expectInterval: expectInterval,
		lastPing:       time.Now().Add(-expectInterval),
	}
}

func (c *ConnChk) Ping() {
	c.lastPing = time.Now()
}

func (c *ConnChk) Check() error {
	if time.Since(c.lastPing) > c.expectInterval+c.gracePeriod {
		return errors.New("no ping received during the expected interval")
	}
	return nil
}
