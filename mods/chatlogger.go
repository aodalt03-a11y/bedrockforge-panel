package mods

import (
"fmt"
"time"
"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type ChatLogger struct {
enabled bool
}

func init() { Register(&ChatLogger{enabled: true}) }

func (c *ChatLogger) Name() string           { return "ChatLogger" }
func (c *ChatLogger) Init(_ *Context, _ any) { fmt.Println("[ChatLogger] ready") }
func (c *ChatLogger) Tick()                  {}

func (c *ChatLogger) HandlePacket(ctx *Context) {
if !c.enabled { return }
if ctx.Direction != FromServer { return }
pk, ok := ctx.Packet.(*packet.Text)
if !ok { return }
fmt.Printf("[ChatLogger %s] <%s> %s\n", time.Now().Format("15:04:05"), pk.SourceName, pk.Message)
}
