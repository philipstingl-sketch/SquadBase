package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// The Spotify developer dashboard currently accepts local HTTP loopback
// callbacks with an explicit port more reliably than port-less callbacks.
// The app UI continues using a random port. Only OAuth has a stable port.
const spotifyCallbackURI = "http://127.0.0.1:8888/spotify/callback"
const spotifyCallbackAddress = "127.0.0.1:8888"

var spotifyCallbackMu sync.Mutex

func startSpotifyCallbackServer(s *SpotifyEngine) (*http.Server, error) {
	spotifyCallbackMu.Lock()
	defer spotifyCallbackMu.Unlock()
	listener, err := net.Listen("tcp4", spotifyCallbackAddress)
	if err != nil {
		return nil, fmt.Errorf("Port 8888 belegt oder nicht verfügbar: %w", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/spotify/callback", s.callback)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	s.mu.Lock()
	s.redirectURI = spotifyCallbackURI
	s.callbackError = ""
	s.mu.Unlock()
	go func() { _ = server.Serve(listener) }()
	return server, nil
}
