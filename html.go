package main

const indexHTML = `<!DOCTYPE html>
<html><head><title>BedrockForge Panel</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#0a0a0f;color:#00ffcc;font-family:monospace;padding:20px}
h1{margin-bottom:20px}
.card{background:#111;border:1px solid #1a1a2e;border-radius:8px;padding:16px;margin-bottom:16px}
input{background:#0a0a0f;border:1px solid #333;color:#fff;padding:10px;border-radius:6px;width:100%;margin-bottom:10px;font-family:monospace}
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
  const s=document.getElementById("srv").value;
  const r=await fetch("/api/start",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({Server:s})});
  const t=await r.text();
  if(t!="started")alert("Error: "+t);
}
async function stop(){await fetch("/api/stop",{method:"POST",body:"{}"});}
async function tick(){
  try{
    const s=await fetch("/api/status").then(r=>r.json());
    document.getElementById("status").innerHTML="Status: <span class="+(s.running?"on":"off")+">"+(s.running?"Online":"Offline")+"</span>";
    const l=await fetch("/api/logs").then(r=>r.json());
    const log=document.getElementById("log");
    log.innerHTML=l.map(x=>"<div style=color:#00ffcc88>"+x+"</div>").join("");
    log.scrollTop=log.scrollHeight;
    if(!window._authOpened){
      const a=l.find(x=>x.includes("microsoft.com/link"));
      if(a){
        const m=a.match(/https:\/\/\S+/);
        if(m){window._authOpened=true;window.open(m[0],"_blank");}
      }
    }
  }catch(e){}
}
setInterval(tick,2000);tick();
</script></body></html>`
