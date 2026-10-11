package main

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMinecraftStatusParser(t *testing.T) {
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() {
		c, e := l.Accept()
		if e != nil {
			return
		}
		defer c.Close()
		_ = c.SetDeadline(time.Now().Add(5 * time.Second))
		for i := 0; i < 2; i++ {
			n, e := readVarInt(c)
			if e != nil {
				return
			}
			_, e = io.CopyN(io.Discard, c, int64(n))
			if e != nil {
				return
			}
		}
		data := []byte(`{"version":{"name":"1.21.1"},"players":{"online":7,"max":20}}`)
		answer := append([]byte{0}, writeVarInt(len(data))...)
		answer = append(answer, data...)
		_ = writePacket(c, answer)
	}()
	s := probeMinecraft(l.Addr().String())
	if !s.Online || s.Players != 7 || s.MaxPlayers != 20 || s.Version != "1.21.1" {
		t.Fatalf("incorrect Minecraft status: %+v", s)
	}
}
func TestDashboardAssetAndState(t *testing.T) {
	app := &App{cfg: Config{Server: "play.hugosmp.de", GamePaths: map[string]string{}}, usage: map[string]int64{}, playing: map[string]bool{}, prefix: "/s/testtoken/", lastSeen: time.Now()}
	for _, name := range []string{"", "style.css", "app.js", "api/state"} {
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:9988/s/testtoken/"+name, nil)
		req.RemoteAddr = "127.0.0.1:34321"
		w := httptest.NewRecorder()
		app.handle(w, req)
		if w.Code != 200 {
			t.Errorf("route %s failed: %d %s", name, w.Code, w.Body.String())
		}
		if name == "api/state" {
			var body map[string]any
			if json.Unmarshal(w.Body.Bytes(), &body) != nil || body["config"] == nil {
				t.Fatal("state JSON invalid")
			}
		}
	}
}
func TestPostRequiresTokenHeader(t *testing.T) {
	app := &App{prefix: "/s/token/", lastSeen: time.Now()}
	r := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:9988/s/token/api/speed", strings.NewReader(`{}`))
	r.RemoteAddr = "127.0.0.1:55555"
	w := httptest.NewRecorder()
	app.handle(w, r)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}
