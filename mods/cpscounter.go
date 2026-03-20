package mods

import (
"fmt"
"sync/atomic"
"time"
"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type CPSCounter struct {
clicks    int64
lastPrint time.Time
}

func init() {}

func (c *CPSCounter) Name() string { return "CPSCounter" }
func (c *CPSCounter) Init(_ *Context, _ any) {
c.lastPrint = time.Now()
fmt.Println("[CPSCounter] ready")
}

func (c *CPSCounter) HandlePacket(ctx *Context) {
if ctx.Direction != FromClient {
return
}
if _, ok := ctx.Packet.(*packet.InventoryTransaction); ok {
atomic.AddInt64(&c.clicks, 1)
}
}

func (c *CPSCounter) Tick() {
if time.Since(c.lastPrint) >= time.Second {
fmt.Printf("[CPSCounter] %d CPS\n", atomic.SwapInt64(&c.clicks, 0))
c.lastPrint = time.Now()
}
}
