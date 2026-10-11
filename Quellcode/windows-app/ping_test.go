package main

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestPingEndpoint(t *testing.T) {
	cases := []struct{ choice, custom, minecraft, want string }{
		{"cloudflare", "", "play.hugosmp.de", "1.1.1.1:443"},
		{"google", "", "play.hugosmp.de", "8.8.8.8:443"},
		{"quad9", "", "play.hugosmp.de", "9.9.9.9:853"},
		{"minecraft", "", "play.hugosmp.de", "play.hugosmp.de:25565"},
		{"minecraft", "", "play.hugosmp.de:25570", "play.hugosmp.de:25570"},
		{"custom", "example.org:443", "", "example.org:443"},
		{"custom", "1.1.1.1", "", "1.1.1.1:443"},
	}
	for _, tc := range cases {
		_, got, err := pingEndpoint(tc.choice, tc.custom, tc.minecraft)
		if err != nil || got != tc.want {
			t.Errorf("%s %q: got %q, %v, want %q", tc.choice, tc.custom, got, err, tc.want)
		}
	}
}
func TestPingEndpointRejectsBadInputs(t *testing.T) {
	for _, host := range []string{"http://example.org", "example.org:0", "example.org:65536", "bad/host", "", "www..example.com", "host && whoami", "example.org:notaport"} {
		if _, _, err := pingEndpoint("custom", host, ""); err == nil {
			t.Errorf("accepted %q", host)
		}
	}
	if _, _, err := pingEndpoint("invalid", "", ""); err == nil {
		t.Fatal("accepted invalid preset")
	}
}
func TestPingMeasuresRealLocalTCP(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			c, e := listener.Accept()
			if e != nil {
				return
			}
			_ = c.Close()
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	result := measureTargetPing(ctx, "Test-Server", listener.Addr().String())
	if result.Successful != 5 || result.Average == nil || *result.Average < 0 || result.Running {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !strings.Contains(result.Endpoint, "127.0.0.1") {
		t.Fatalf("wrong endpoint %+v", result)
	}
}
