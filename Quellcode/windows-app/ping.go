package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// TargetPing measures TCP handshake latency. This is not an ICMP ping and
// must not be presented as a game's own server latency.
type TargetPing struct {
	Running    bool     `json:"running"`
	Label      string   `json:"label"`
	Endpoint   string   `json:"endpoint"`
	Average    *float64 `json:"average"`
	Minimum    *float64 `json:"minimum"`
	Maximum    *float64 `json:"maximum"`
	Successful int      `json:"successful"`
	Attempts   int      `json:"attempts"`
	Error      string   `json:"error"`
	TestedAt   string   `json:"testedAt"`
}

func parsePingHost(input string, defaultPort string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" || len(input) > 255 || strings.ContainsAny(input, "/\\?#@ \t\r\n") {
		return "", errors.New("Bitte Hostname oder IP-Adresse eingeben (ohne http://)")
	}
	host, port, e := net.SplitHostPort(input)
	if e != nil {
		if strings.Count(input, ":") > 0 {
			if net.ParseIP(input) == nil {
				return "", errors.New("Ungültiger Host oder Port")
			}
			host = input
		} else {
			host = input
		}
		port = defaultPort
	}
	if host == "" {
		return "", errors.New("Hostname fehlt")
	}
	if net.ParseIP(host) == nil {
		if len(host) > 253 {
			return "", errors.New("Hostname zu lang")
		}
		host = strings.TrimSuffix(host, ".")
		for _, part := range strings.Split(host, ".") {
			if len(part) == 0 || len(part) > 63 || part[0] == '-' || part[len(part)-1] == '-' {
				return "", errors.New("Ungültiger Hostname")
			}
			for _, c := range part {
				if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '-' {
					return "", errors.New("Ungültiger Hostname")
				}
			}
		}
	}
	n, e := strconv.Atoi(port)
	if e != nil || n < 1 || n > 65535 {
		return "", errors.New("Port muss zwischen 1 und 65535 liegen")
	}
	return net.JoinHostPort(host, strconv.Itoa(n)), nil
}

func pingEndpoint(choice, custom, minecraft string) (string, string, error) {
	switch choice {
	case "cloudflare":
		return "Cloudflare DNS", "1.1.1.1:443", nil
	case "google":
		return "Google DNS", "8.8.8.8:443", nil
	case "quad9":
		return "Quad9 DNS", "9.9.9.9:853", nil
	case "minecraft":
		h, e := parsePingHost(minecraft, "25565")
		if e != nil {
			return "", "", e
		}
		return "Dein Minecraft-Server", h, nil
	case "custom":
		h, e := parsePingHost(custom, "443")
		if e != nil {
			return "", "", e
		}
		return "Eigener Server", h, nil
	default:
		return "", "", fmt.Errorf("Unbekanntes Ping-Ziel")
	}
}

func measureTargetPing(ctx context.Context, label, endpoint string) TargetPing {
	result := TargetPing{Running: true, Label: label, Endpoint: endpoint, Attempts: 5}
	var vals []float64
	for i := 0; i < result.Attempts; i++ {
		if ctx.Err() != nil {
			break
		}
		attemptCtx, cancel := context.WithTimeout(ctx, 1800*time.Millisecond)
		start := time.Now()
		conn, e := (&net.Dialer{}).DialContext(attemptCtx, "tcp", endpoint)
		if e == nil {
			_ = conn.Close()
			vals = append(vals, float64(time.Since(start).Microseconds())/1000)
		}
		cancel()
		if i+1 < result.Attempts {
			select {
			case <-ctx.Done():
				break
			case <-time.After(180 * time.Millisecond):
			}
		}
	}
	result.Successful = len(vals)
	if len(vals) > 0 {
		lo, hi, sum := vals[0], vals[0], 0.0
		for _, v := range vals {
			sum += v
			if v < lo {
				lo = v
			}
			if v > hi {
				hi = v
			}
		}
		avg := sum / float64(len(vals))
		result.Average = &avg
		result.Minimum = &lo
		result.Maximum = &hi
		if len(vals) < result.Attempts {
			result.Error = "Einige Verbindungen sind fehlgeschlagen"
		}
	} else {
		result.Error = "Keine TCP-Verbindung möglich. Der Host/Port blockiert eventuell Verbindungen oder ist offline."
	}
	result.Running = false
	result.TestedAt = time.Now().Format(time.RFC3339)
	return result
}

func (a *App) runPing(label, endpoint string) {
	ctx, cancel := context.WithTimeout(context.Background(), 16*time.Second)
	defer cancel()
	result := measureTargetPing(ctx, label, endpoint)
	a.mu.Lock()
	a.ping = result
	a.mu.Unlock()
}
