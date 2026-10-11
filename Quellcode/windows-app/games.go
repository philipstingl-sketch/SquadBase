package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

func activeProcesses() map[string]bool {
	on := map[string]bool{}
	ctx, c := context.WithTimeout(context.Background(), 8*time.Second)
	defer c()
	b, e := hiddenOutput(ctx, "tasklist.exe", "/FO", "CSV", "/NH")
	if e != nil {
		return on
	}
	rows, e := csv.NewReader(strings.NewReader(string(b))).ReadAll()
	if e != nil {
		return on
	}
	present := map[string]bool{}
	for _, row := range rows {
		if len(row) > 0 {
			present[strings.ToLower(strings.TrimSpace(row[0]))] = true
		}
	}
	names := map[string][]string{"minecraft": {"minecraft.windows.exe"}, "rocket": {"rocketleague.exe"}, "gta": {"gta5.exe", "gta5_enhanced.exe", "playgtav.exe"}, "valorant": {"valorant-win64-shipping.exe"}, "fortnite": {"fortniteclient-win64-shipping.exe"}}
	for k, ps := range names {
		for _, p := range ps {
			if present[p] {
				on[k] = true
			}
		}
	}
	// Javaw.exe is shared by many apps. Count it only if its Windows command line identifies Minecraft.
	if present["javaw.exe"] && !on["minecraft"] {
		javaCtx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		out, err := hiddenOutput(javaCtx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", `Get-CimInstance Win32_Process -Filter "Name='javaw.exe'" | Select-Object -ExpandProperty CommandLine`)
		if err == nil {
			line := strings.ToLower(string(out))
			if strings.Contains(line, "--gamedir") || strings.Contains(line, "net.minecraft") || strings.Contains(line, "fabric-loader") || strings.Contains(line, "minecraftlauncher") {
				on["minecraft"] = true
			}
		}
		cancel()
	}
	return on
}
func writeVarInt(n int) []byte {
	b := []byte{}
	for {
		if n&^0x7f == 0 {
			return append(b, byte(n))
		}
		b = append(b, byte(n&0x7f|0x80))
		n >>= 7
	}
}
func readVarInt(r io.Reader) (int, error) {
	out := 0
	for i := 0; i < 5; i++ {
		var b [1]byte
		if _, e := io.ReadFull(r, b[:]); e != nil {
			return 0, e
		}
		out |= int(b[0]&0x7f) << (i * 7)
		if b[0]&128 == 0 {
			return out, nil
		}
	}
	return 0, errors.New("varint invalid")
}
func readVarIntSlice(b []byte, p *int) (int, error) {
	v := 0
	for i := 0; i < 5; i++ {
		if *p >= len(b) {
			return 0, io.EOF
		}
		x := b[*p]
		*p = *p + 1
		v |= int(x&127) << (7 * i)
		if x&128 == 0 {
			return v, nil
		}
	}
	return 0, errors.New("varint invalid")
}
func writePacket(w io.Writer, data []byte) error {
	payload := append(writeVarInt(len(data)), data...)
	_, e := w.Write(payload)
	return e
}
func probeMinecraft(raw string) ServerStatus {
	st := ServerStatus{CheckedAt: time.Now().Format(time.RFC3339)}
	host, port, err := net.SplitHostPort(raw)
	if err != nil {
		host = raw
		port = "25565"
	}
	portN, err := strconv.Atoi(port)
	if err != nil || portN < 1 || portN > 65535 {
		st.Error = "Ungültiger Port"
		return st
	}
	// Java Edition may advertise a different port through _minecraft._tcp SRV.
	connectHost := host
	if !strings.Contains(raw, ":") {
		_, records, srvErr := net.LookupSRV("minecraft", "tcp", host)
		if srvErr == nil && len(records) > 0 {
			connectHost = strings.TrimSuffix(records[0].Target, ".")
			portN = int(records[0].Port)
			port = strconv.Itoa(portN)
		}
	}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(connectHost, port), 4*time.Second)
	if err != nil {
		st.Error = "Nicht erreichbar"
		return st
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	st.Latency = time.Since(start).Milliseconds()
	req := []byte{0}
	req = append(req, writeVarInt(767)...)
	req = append(req, writeVarInt(len(host))...)
	req = append(req, []byte(host)...)
	req = append(req, byte(portN>>8), byte(portN), 1)
	if writePacket(conn, req) != nil || writePacket(conn, []byte{0}) != nil {
		st.Error = "Anfrage fehlgeschlagen"
		return st
	}
	n, e := readVarInt(conn)
	if e != nil || n < 2 || n > 2_000_000 {
		st.Error = "Keine gültige Serverantwort"
		return st
	}
	b := make([]byte, n)
	if _, e = io.ReadFull(conn, b); e != nil {
		st.Error = "Statusdaten unvollständig"
		return st
	}
	p := 0
	packet, e := readVarIntSlice(b, &p)
	if e != nil || packet != 0 {
		st.Error = "Statusantwort ungültig"
		return st
	}
	length, e := readVarIntSlice(b, &p)
	if e != nil || length < 0 || p+length > len(b) {
		st.Error = "Statusantwort ungültig"
		return st
	}
	var body struct {
		Version struct {
			Name string `json:"name"`
		} `json:"version"`
		Players struct {
			Online int `json:"online"`
			Max    int `json:"max"`
		} `json:"players"`
	}
	if json.Unmarshal(b[p:p+length], &body) != nil {
		st.Error = "Statusantwort nicht lesbar"
		return st
	}
	st.Online = true
	st.Players = body.Players.Online
	st.MaxPlayers = body.Players.Max
	st.Version = body.Version.Name
	return st
}
