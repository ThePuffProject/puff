package puff

import (
	"github.com/tiredkangaroo/websocket"
)

// handleWebSocket accepts a new WebSocket connection and initializes the WebSocket struct.
func (c *Context) handleWebSocket() error {
	conn, err := websocket.AcceptHTTP(c.ResponseWriter, c.Request)
	if err != nil {
		c.BadRequest(err.Error())
		return err
	}

	c.WebSocket = conn
	return nil
}
