package puff_test

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/ThePuffProject/puff"
	"github.com/tiredkangaroo/websocket"
)

func TestApp(t *testing.T) {
	// Test with all configuration fields set
	config := puff.AppConfig{
		Name:              "TestApp",
		Version:           "1.2.3",
		DocsURL:           "/test-docs",
		TLSPublicCertFile: "cert.pem",
		TLSPrivateKeyFile: "key.pem",
	}
	app := puff.App(&config)

	if app.Config.Name != "TestApp" {
		t.Errorf("Expected app name 'TestApp', got '%s'", app.Config.Name)
	}
	if app.Config.Version != "1.2.3" {
		t.Errorf("Expected version '1.2.3', got '%s'", app.Config.Version)
	}
	if app.Config.DocsURL != "/test-docs" {
		t.Errorf("Expected DocsURL '/test-docs', got '%s'", app.Config.DocsURL)
	}
	if app.Config.TLSPublicCertFile != "cert.pem" {
		t.Errorf("Expected TLSPublicCertFile 'cert.pem', got '%s'", app.Config.TLSPublicCertFile)
	}
	if app.Config.TLSPrivateKeyFile != "key.pem" {
		t.Errorf("Expected TLSPrivateKeyFile 'key.pem', got '%s'", app.Config.TLSPrivateKeyFile)
	}
	if app.Config.OpenAPI != nil {
		t.Errorf("Expected OpenAPI to not be set.")
	}
	if app.rootRouter == nil {
		t.Fatalf("Expected rootRouter to be initialized")
	}
	if app.rootRouter.Name != "Default" {
		t.Errorf("Expected rootRouter name 'Default', got '%s'", app.rootRouter.Name)
	}
	if app.rootRouter.Tag != "Default" {
		t.Errorf("Expected rootRouter tag 'Default', got '%s'", app.rootRouter.Tag)
	}
}

func TestApp_DefaultVersion(t *testing.T) {
	// Test with default version when version is not provided in the config
	config := puff.AppConfig{
		Name: "TestAppWithDefaultVersion",
	}
	app := puff.App(&config)

	if app.Config.Version != "0.0.0" {
		t.Errorf("Expected default version '0.0.0', got '%s'", app.Config.Version)
	}
}

func TestDefaultApp(t *testing.T) {
	app := puff.DefaultApp("DefaultAppTest")

	if app.Config.Name != "DefaultAppTest" {
		t.Errorf("Expected app name 'DefaultAppTest', got '%s'", app.Config.Name)
	}
	if app.Config.Version != "1.0.0" {
		t.Errorf("Expected default version '1.0.0', got '%s'", app.Config.Version)
	}
	if app.Config.DocsURL != "/docs" {
		t.Errorf("Expected default DocsURL '/docs', got '%s'", app.Config.DocsURL)
	}
}

// testpuffserver starts a puff server for testing. It panics
// if the server is unavailable.
func testpuffserver() {
	app := puff.DefaultApp("")

	app.Get("/test", nil, func(ctx *puff.Context) {
		ctx.SendResponse(puff.GenericResponse{
			StatusCode:  200,
			Content:     "hello world",
			ContentType: "text/plain",
		})
	})

	app.Post("/data", nil, func(ctx *puff.Context) {
		data, err := ctx.GetBody()
		if err != nil {
			ctx.SendResponse(puff.GenericResponse{
				StatusCode:  500,
				Content:     err.Error(),
				ContentType: "text/plain",
			})
		}
		b := sha1.Sum(append(data, data...))
		ctx.SendResponse(puff.GenericResponse{
			StatusCode:  200,
			Content:     hex.EncodeToString(b[:]),
			ContentType: "text/plain",
		})
	})

	app.Get("/json", nil, func(ctx *puff.Context) {
		ctx.SendResponse(puff.JSONResponse{
			StatusCode: 200,
			Content: map[string]any{
				"fuzzabc": "cbazzuf",
			},
		})
	})

	app.WebSocket("/ws", nil, func(c *puff.Context) {
		c.WebSocket.Write(&websocket.Message{
			Type: websocket.MessageText,
			Data: []byte("hello world!"),
		})
	})

	go func() {
		app.ListenAndServe(":7465")
	}()

	time.Sleep(time.Second * 2)
	_, err := http.Get("http://127.0.0.1:7465/test")
	if err != nil {
		panic(err)
	}
}

// testnethttpserver starts a net/http server for testing. It panics
// if the server is unavailable.
func testnethttpserver() {
	http.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "hello world")
	})

	http.HandleFunc("POST /data", func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/plain")
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
		b := sha1.Sum(append(data, data...))
		fmt.Fprintf(w, hex.EncodeToString(b[:]))
	})

	go func() {
		http.ListenAndServe(":7467", nil)
	}()

	time.Sleep(time.Second * 2)
	_, err := http.Get("http://127.0.0.1:7467/test")
	if err != nil {
		panic(err)
	}
}

func randomdatagen() []byte {
	data := make([]byte, 5e6)
	rand.Read(data)
	return data
}

var (
	oncepuffserver    = sync.OnceFunc(testpuffserver)
	oncenethttpserver = sync.OnceFunc(testnethttpserver)
	randomdata        = sync.OnceValue(randomdatagen)
)
