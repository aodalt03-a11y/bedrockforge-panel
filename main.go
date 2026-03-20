package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"bedrockforge/config"
	"bedrockforge/proxy"
)

func init() {
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{}
		return d.DialContext(ctx, "udp", "8.8.8.8:53")
	}}
}

var botRunning bool
var botMu sync.Mutex
var logs []string
var logMu sync.Mutex

func addLog(msg string) {
	msg = strings.TrimSpace(msg)
	if msg == "" { return }
	logMu.Lock()
	logs = append(logs, time.Now().Format("15:04:05")+" "+msg)
	if len(logs) > 200 { logs = logs[len(logs)-200:] }
	logMu.Unlock()
	os.Stderr.WriteString(msg)
}

func main() {
	pr, pw, _ := os.Pipe()
	os.Stdout = pw
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := pr.Read(buf)
			if n > 0 { addLog(string(buf[:n])) }
			if err != nil { break }
		}
	}()
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/start", handleStart)
	http.HandleFunc("/api/stop", handleStop)
	http.HandleFunc("/api/logs", handleLogs)
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	addLog("Panel starting on :"+port)
	http.ListenAndServe(":"+port, nil)
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	cors(w)
	botMu.Lock(); running := botRunning; botMu.Unlock()
	json.NewEncoder(w).Encode(map[string]interface{}{"running": running})
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	cors(w)
	logMu.Lock(); l := make([]string, len(logs)); copy(l, logs); logMu.Unlock()
	json.NewEncoder(w).Encode(l)
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != http.MethodPost { http.Error(w, "POST only", 405); return }
	var req struct { Server string }
	json.NewDecoder(r.Body).Decode(&req)
	if req.Server == "" { req.Server = os.Getenv("MC_SERVER") }
	if req.Server == "" { http.Error(w, "server required", 400); return }
	botMu.Lock()
	if botRunning { botMu.Unlock(); http.Error(w, "already running", 409); return }
	botRunning = true
	botMu.Unlock()
	addLog("Starting bot -> "+req.Server)
	go func() {
		cfg := &config.Config{ServerAddr: req.Server, ListenAddr: "0.0.0.0:19132", Headless: true}
		proxy.StartHeadlessAndListen(cfg)
		botMu.Lock(); botRunning = false; botMu.Unlock()
		addLog("Bot stopped")
	}()
	fmt.Fprint(w, "started")
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	cors(w)
	botMu.Lock(); botRunning = false; botMu.Unlock()
	fmt.Fprint(w, "stopped")
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, indexHTML)
}
