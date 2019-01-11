package main

import (
	"context"
	"encoding/json"
	"fmt"
	. "github.com/aandryashin/matchers"
	"github.com/gorilla/websocket"
	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/rpcc"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var (
	srv         *httptest.Server
	devtoolsSrv *httptest.Server
)

func init() {
	srv = httptest.NewServer(ws())
	listen = srv.Listener.Addr().String()
	devtoolsSrv = httptest.NewServer(mockDevtoolsMux())
	devtoolsHost = devtoolsSrv.Listener.Addr().String()
}

func mockDevtoolsMux() http.Handler {
	mux := http.NewServeMux()
	targets := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`[ {
		"description": "",
		"devtoolsFrontendUrl": "/devtools/inspector.html?ws=%s/devtools/page/DAB7FB6187B554E10B0BD18821265734",
		"id": "DAB7FB6187B554E10B0BD18821265734",
		"title": "Yahoo",
		"type": "page",
		"url": "https://www.yahoo.com/",
		"webSocketDebuggerUrl": "ws://%s/devtools/page/DAB7FB6187B554E10B0BD18821265734"
		} ]`, devtoolsHost, devtoolsHost)))
	}
	mux.HandleFunc("/json", targets)
	mux.HandleFunc("/json/list", targets)
	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
	}
	mux.HandleFunc("/devtools/page/", func(w http.ResponseWriter, r *http.Request) {
		//Echo request ID but omit Method
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			panic(err)
		}
		defer c.Close()
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			type req struct {
				ID uint64 `json:"id"`
			}
			var r req
			err = json.Unmarshal(message, &r)
			if err != nil {
				panic(err)
			}
			output, err := json.Marshal(r)
			if err != nil {
				panic(err)
			}
			err = c.WriteMessage(mt, output)
			if err != nil {
				break
			}
		}
	})
	return mux
}

func TestDevtools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	u := fmt.Sprintf("ws://%s/", srv.Listener.Addr().String())
	conn, err := rpcc.DialContext(ctx, u)
	AssertThat(t, err, Is{nil})
	defer conn.Close()

	c := cdp.NewClient(conn)
	err = c.Page.Enable(ctx)
	AssertThat(t, err, Is{nil})
}
