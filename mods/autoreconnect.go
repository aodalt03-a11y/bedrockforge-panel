package mods

import (
"fmt"
"time"
"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const ReconnectDelay = 5 * time.Second

type AutoReconnect struct {
ReconnectCh chan struct{}
triggered   bool
}

func init() { Register(&AutoReconnect{ReconnectCh: make(chan struct{}, 1)}) }

func (a *AutoReconnect) Name() string           { return "AutoReconnect" }
func (a *AutoReconnect) Init(_ *Context, _ any) { fmt.Println("[AutoReconnect] ready") }
func (a *AutoReconnect) Tick()                  {}

func (a *AutoReconnect) HandlePacket(ctx *Context) {
if ctx.Direction != FromServer {
return
}
pk, ok := ctx.Packet.(*packet.Disconnect)
if !ok || a.triggered {
return
}
a.triggered = true
fmt.Printf("[AutoReconnect] disconnected (%s) - reconnecting in %s\n", pk.Message, ReconnectDelay)
go func() {
time.Sleep(ReconnectDelay)
a.triggered = false
a.ReconnectCh <- struct{}{}
}()
}
