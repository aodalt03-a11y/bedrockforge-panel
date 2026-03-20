package mods

import (
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type Direction int

const (
	FromClient Direction = iota
	FromServer
)

type Context struct {
	Packet       packet.Packet
	Direction    Direction
	Drop         bool
	SendToClient func(pk packet.Packet) error
	SendToServer func(pk packet.Packet) error
}

type Mod interface {
	Name() string
	Init(ctx *Context, config any)
	HandlePacket(ctx *Context)
	Tick()
}

var (
	mu       sync.RWMutex
	registry []Mod
)

func Register(m Mod) {
	mu.Lock()
	defer mu.Unlock()
	registry = append(registry, m)
}

func All() []Mod {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Mod, len(registry))
	copy(out, registry)
	return out
}
