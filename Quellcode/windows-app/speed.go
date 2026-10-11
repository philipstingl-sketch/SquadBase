package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Real HTTP network measurement against Cloudflare's public speed-test endpoints.
// Unlike an ICMP ping, the latency is HTTP/TLS response latency to the chosen service.
func (a *App) speedTest() {
	s := SpeedResult{Running: true, Phase: "Ping zu Cloudflare"}
	defer func() { s.Running = false; s.TestedAt = time.Now().Format(time.RFC3339); a.setSpeed(s) }()
	cl := &http.Client{Timeout: 25 * time.Second, Transport: &http.Transport{DisableKeepAlives: true, ResponseHeaderTimeout: 10 * time.Second}}
	var ms []float64
	for i := 0; i < 4; i++ {
		ctx, c := context.WithTimeout(context.Background(), 8*time.Second)
		req, _ := http.NewRequestWithContext(ctx, "GET", "https://speed.cloudflare.com/__down?bytes=0", nil)
		start := time.Now()
		res, e := cl.Do(req)
		if e == nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 256))
			_ = res.Body.Close()
			if res.StatusCode == 200 {
				ms = append(ms, time.Since(start).Seconds()*1000)
			}
		}
		c()
	}
	if len(ms) > 0 {
		sum := 0.0
		for _, x := range ms {
			sum += x
		}
		ping := sum / float64(len(ms))
		s.Ping = &ping
	}
	s.Phase = "Download messen"
	a.setSpeed(s)
	down, e := downloadTest(cl)
	if e != nil {
		s.Error = "Download fehlgeschlagen: " + e.Error()
		return
	}
	s.Download = &down
	s.Phase = "Upload messen"
	a.setSpeed(s)
	up, e := uploadTest(cl)
	if e != nil {
		s.Error = "Upload fehlgeschlagen: " + e.Error()
		return
	}
	s.Upload = &up
	s.Phase = "Messung abgeschlossen"
}
func downloadTest(cl *http.Client) (float64, error) {
	sizes := []int{2_000_000, 6_000_000, 10_000_000}
	var total int64
	var elapsed time.Duration
	for _, size := range sizes {
		ctx, c := context.WithTimeout(context.Background(), 25*time.Second)
		req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://speed.cloudflare.com/__down?bytes=%d&ts=%d", size, time.Now().UnixNano()), nil)
		start := time.Now()
		res, e := cl.Do(req)
		if e != nil {
			c()
			return 0, errors.New("Cloudflare nicht erreichbar")
		}
		if res.StatusCode != 200 {
			_ = res.Body.Close()
			c()
			return 0, fmt.Errorf("HTTP %d", res.StatusCode)
		}
		n, e := io.Copy(io.Discard, io.LimitReader(res.Body, int64(size)+1000))
		duration := time.Since(start)
		_ = res.Body.Close()
		c()
		if e != nil || n < int64(size) {
			return 0, errors.New("Übertragung unvollständig")
		}
		total += n
		elapsed += duration
		if elapsed > 6*time.Second {
			break
		}
	}
	if elapsed <= 0 {
		return 0, errors.New("Keine Messdaten")
	}
	return float64(total) * 8 / elapsed.Seconds() / 1e6, nil
}
func uploadTest(cl *http.Client) (float64, error) {
	sizes := []int{1_000_000, 3_000_000, 5_000_000}
	buf := make([]byte, 5_000_000)
	if _, e := rand.Read(buf); e != nil {
		return 0, e
	}
	var total int64
	var elapsed time.Duration
	for _, size := range sizes {
		ctx, c := context.WithTimeout(context.Background(), 25*time.Second)
		req, _ := http.NewRequestWithContext(ctx, "POST", "https://speed.cloudflare.com/__up", bytes.NewReader(buf[:size]))
		req.Header.Set("Content-Type", "application/octet-stream")
		start := time.Now()
		res, e := cl.Do(req)
		if e != nil {
			c()
			return 0, errors.New("Cloudflare-Upload nicht erreichbar")
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 2048))
		_ = res.Body.Close()
		duration := time.Since(start)
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			c()
			return 0, fmt.Errorf("HTTP %d", res.StatusCode)
		}
		c()
		total += int64(size)
		elapsed += duration
		if elapsed > 6*time.Second {
			break
		}
	}
	if elapsed <= 0 {
		return 0, errors.New("Keine Messdaten")
	}
	return float64(total) * 8 / elapsed.Seconds() / 1e6, nil
}
