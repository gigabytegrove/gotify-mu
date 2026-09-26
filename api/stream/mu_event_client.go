package stream

import (
	"time"

	"github.com/gorilla/websocket"
)

type muEventClient struct {
	conn    *websocket.Conn
	onClose func(*muEventClient)
	write   chan any
	closed  chan struct{}
	userID  uint
	token   string
	once    once
}

func newMUEventClient(
	conn *websocket.Conn,
	userID uint,
	token string,
	onClose func(*muEventClient),
) *muEventClient {
	return &muEventClient{
		conn:    conn,
		onClose: onClose,
		write:   make(chan any, 8),
		closed:  make(chan struct{}),
		userID:  userID,
		token:   token,
	}
}

func (c *muEventClient) Close() {
	c.once.Do(func() {
		c.conn.Close()
		close(c.closed)
	})
}

func (c *muEventClient) NotifyClose() {
	c.once.Do(func() {
		c.conn.Close()
		close(c.closed)
		c.onClose(c)
	})
}

func (c *muEventClient) startReading(pongWait time.Duration) {
	defer c.NotifyClose()
	c.conn.SetReadLimit(256)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(appData string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		if _, _, err := c.conn.NextReader(); err != nil {
			printWebSocketError("MUEventReadError", err)
			return
		}
	}
}

func (c *muEventClient) startWriteHandler(pingPeriod time.Duration) {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		c.NotifyClose()
		pingTicker.Stop()
	}()

	for {
		select {
		case <-c.closed:
			return
		case event := <-c.write:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := writeJSON(c.conn, event); err != nil {
				printWebSocketError("MUEventWriteError", err)
				return
			}
		case <-pingTicker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ping(c.conn); err != nil {
				printWebSocketError("MUEventPingError", err)
				return
			}
		}
	}
}
