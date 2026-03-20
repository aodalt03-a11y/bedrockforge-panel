package mods

import (
"fmt"
"reflect"
"sync"
"time"
)

type PacketLogger struct {
	enabled bool
mu      sync.Mutex
counts  map[string]int
lastOut time.Time
}

func init() {}

func (p *PacketLogger) Name() string           { return "PacketLogger" }
func (p *PacketLogger) Init(_ *Context, _ any) {
p.lastOut = time.Now()
}

func (p *PacketLogger) HandlePacket(ctx *Context) {
if !p.enabled { return }
	// Skip already-dropped packets
if ctx.Drop {
return
}
dir := "S->C"
if ctx.Direction == FromClient {
dir = "C->S"
}
name := reflect.TypeOf(ctx.Packet).Elem().Name()
p.mu.Lock()
p.counts[fmt.Sprintf("[%s] %s", dir, name)]++
p.mu.Unlock()
}

func (p *PacketLogger) Tick() {
p.mu.Lock()
defer p.mu.Unlock()
if time.Since(p.lastOut) < time.Second || len(p.counts) == 0 {
return
}
fmt.Println("[PacketLogger] -- summary --")
for k, v := range p.counts {
fmt.Printf("  %-40s x%d\n", k, v)
}
p.counts = make(map[string]int)
p.lastOut = time.Now()
}
