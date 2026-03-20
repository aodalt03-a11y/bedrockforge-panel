package main

import (
"context"
"encoding/json"
"fmt"
"log"
"net"
"net/http"
"os"
"sync"
"time"

"bedrockforge/config"
"bedrockforge/proxy"
)

func init() {
net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
d := net.Dialer{}; return d.DialContext(ctx, "udp", "8.8.8.8:53")
}}
}

var botRunning bool
var botMu sync.Mutex
var logs []string
var logMu sync.Mutex

func addLog(msg string) {
logMu.Lock()
logs = append(logs, time.Now().Format("15:04:05")+" "+msg)
if len(logs) > 200 { logs = logs[len(logs)-200:] }
logMu.Unlock()
log.Println(msg)
}

func main() {
http.HandleFunc("/", handleIndex)
http.HandleFunc("/api/status", handleStatus)
http.HandleFunc("/api/start", handleStart)
http.HandleFunc("/api/stop", handleStop)
http.HandleFunc("/api/logs", handleLogs)
port := os.Getenv("PORT")
if port == "" { port = "8080" }
addLog("Panel starting on :"+port)
log.Fatal(http.ListenAndServe(":"+port, nil))
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
var req struct { Server string `json:"server"` }
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
addLog("Stop requested")
os.Exit(0)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "text/html")
fmt.Fprint(w, indexHTML)
}

const indexHTML = `<!DOCTYPE html>
<html><head><title>BedrockForge Panel</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#0a0a0f;color:#00ffcc;font-family:monospace;padding:20px}
h1{margin-bottom:20px}
.card{background:#111;border:1px solid #00ffcc33;border-radius:8px;padding:16px;margin-bottom:16px}
input{background:#0a0a0f;border:1px solid #00ffcc55;color:#fff;padding:10px;border-radius:6px;width:100%;margin-bottom:10px;font-family:monospace}
button{padding:10px 20px;border-radius:6px;border:none;cursor:pointer;font-weight:bold;margin-right:8px}
.start{background:#00ffcc;color:#000}.stop{background:#ff4444;color:#fff}
.on{color:#00ffcc}.off{color:#ff4444}
#log{background:#000;border-radius:6px;padding:12px;height:350px;overflow-y:auto;font-size:12px;line-height:1.6}
</style></head><body>
<h1>BedrockForge Bot Panel</h1>
<div class="card">
<div id="status">Status: <span class="off">Offline</span></div><br>
<input id="srv" placeholder="play.netherite.gg:19132">
<button class="start" onclick="start()">Start Bot</button>
<button class="stop" onclick="stop()">Stop Bot</button>
</div>
<div class="card"><div style="margin-bottom:8px;color:#666;font-size:12px">LOG</div>
<div id="log"></div></div>
<script>
async function start(){
  const s=document.getElementById('srv').value;
  await fetch('/api/start',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({server:s})});
}
async function stop(){await fetch('/api/stop',{method:'POST'});}
async function tick(){
  try{
    const s=await fetch('/api/status').then(r=>r.json());
    document.getElementById('status').innerHTML='Status: <span class="'+(s.running?'on':'off')+'">'+(s.running?'Online':'Offline')+'</span>';
    const l=await fetch('/api/logs').then(r=>r.json());
    const log=document.getElementById('log');
    log.innerHTML=l.map(x=>'<div style="color:#00ffcc88">'+x+'</div>').join('');
    log.scrollTop=log.scrollHeight;
    // Auto open auth URL
    const authLine=l.find(x=>x.includes('microsoft.com/link'));
      const url=authLine.match(/(https:\/\/[^\s]+)/);
      if(url){window._authOpened=true;window.open(url[1],'_blank');}
    }
  }catch(e){}
}
setInterval(tick,2000);tick();
</script></body></html>`
