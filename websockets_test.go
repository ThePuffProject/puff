package puff_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/tiredkangaroo/websocket"
)

func TestWebSocket(t *testing.T) {
	oncepuffserver()

	helloworld := []byte("hello world!")
	u, _ := url.Parse("http://127.0.0.1:7465/ws")
	key := "helloworldkey"

	var conn net.Conn
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			c, err := (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext(ctx, network, addr)

			if err == nil {
				conn = c // Capture the connection
			}
			return conn, err
		},
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Ignore TLS for testing
	}
	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Do(&http.Request{
		Method: "GET",
		URL:    u,
		Header: http.Header{
			"Upgrade":               []string{"websocket"},
			"Connection":            []string{"Upgrade"},
			"Sec-WebSocket-Version": []string{"13"},
			"Sec-WebSocket-Key":     []string{key},
		},
	})
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
		t.FailNow()
	}

	if resp.StatusCode != 101 {
		body := make([]byte, 0, 512)
		resp.Body.Read(body)
		t.Errorf("unexpected status code %d response from server, body: %s", resp.StatusCode, string(body))
		t.FailNow()
	}

	if conn == nil {
		t.Errorf("expected conn to not be nil")
		t.FailNow()
	}

	wsconn := websocket.From(conn)
	message, err := wsconn.Read()
	if err != nil {
		t.Errorf("unexpected error reading wsconn: %s", err.Error())
		t.FailNow()
	}
	if !bytes.Equal(message.Data, helloworld) {
		t.Errorf("expected data to be equal, expected: %s, got: %s", string(helloworld), string(message.Data))
		t.FailNow()
	}
}
