package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// cdp is a minimal Chrome DevTools Protocol client. One connection to the
// browser carries every tab, each as a flattened session.
type cdp struct {
	conn    *websocket.Conn
	onEvent func(sessionID, method string, params json.RawMessage)

	wmu     sync.Mutex
	mu      sync.Mutex
	next    int
	pending map[int]chan cdpReply
}

type cdpReply struct {
	result json.RawMessage
	err    error
}

type cdpMessage struct {
	ID        int             `json:"id,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func dialCDP(ctx context.Context, url string, onEvent func(sessionID, method string, params json.RawMessage)) (*cdp, error) {
	d := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := d.DialContext(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("connecting to the browser: %w", err)
	}
	conn.SetReadLimit(64 << 20)
	c := &cdp{conn: conn, onEvent: onEvent, pending: map[int]chan cdpReply{}}
	go c.read()
	return c, nil
}

// read hands replies to their callers and events to onEvent. onEvent must not wait for a reply.
func (c *cdp) read() {
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			for id, ch := range c.pending {
				ch <- cdpReply{err: errors.New("the browser closed")}
				delete(c.pending, id)
			}
			c.mu.Unlock()
			return
		}
		var m cdpMessage
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		if m.ID == 0 {
			if c.onEvent != nil {
				c.onEvent(m.SessionID, m.Method, m.Params)
			}
			continue
		}
		c.mu.Lock()
		ch := c.pending[m.ID]
		delete(c.pending, m.ID)
		c.mu.Unlock()
		if ch == nil {
			continue
		}
		if m.Error != nil {
			ch <- cdpReply{err: fmt.Errorf("%s", m.Error.Message)}
		} else {
			ch <- cdpReply{result: m.Result}
		}
	}
}

// call runs a protocol method, in the given tab session or, with "", on the browser.
func (c *cdp) call(ctx context.Context, sessionID, method string, params any) (json.RawMessage, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("encoding %s: %w", method, err)
	}
	ch := make(chan cdpReply, 1)
	c.mu.Lock()
	c.next++
	id := c.next
	c.pending[id] = ch
	c.mu.Unlock()

	msg, _ := json.Marshal(cdpMessage{ID: id, SessionID: sessionID, Method: method, Params: raw})
	c.wmu.Lock()
	err = c.conn.WriteMessage(websocket.TextMessage, msg)
	c.wmu.Unlock()
	if err != nil {
		c.forget(id)
		return nil, fmt.Errorf("sending %s: %w", method, err)
	}
	select {
	case r := <-ch:
		if r.err != nil {
			return nil, fmt.Errorf("%s: %w", method, r.err)
		}
		return r.result, nil
	case <-ctx.Done():
		c.forget(id)
		return nil, fmt.Errorf("%s: %w", method, ctx.Err())
	}
}

func (c *cdp) forget(id int) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

func (c *cdp) close() { _ = c.conn.Close() }
