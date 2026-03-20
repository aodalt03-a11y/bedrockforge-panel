package mods

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type PacketRecord struct {
	Time      string
	Direction string
	Type      string
	Data      any
}

type PacketRecorder struct {
	encoder *json.Encoder
	file    *os.File
}

func init() { Register(&PacketRecorder{}) }

func (p *PacketRecorder) Name() string { return "PacketRecorder" }
func (p *PacketRecorder) Tick()        {}

func (p *PacketRecorder) Init(ctx *Context, _ any) {
	f, err := os.Create("packet_record.json")
	if err != nil {
		fmt.Println("[Recorder] failed to create file:", err)
		return
	}
	p.file = f
	p.encoder = json.NewEncoder(f)
	p.encoder.SetIndent("", "  ")
	fmt.Println("[Recorder] recording ALL packets to packet_record.json")
}

func (p *PacketRecorder) record(dir string, pk packet.Packet) {
	if p.encoder == nil {
		return
	}
	rec := PacketRecord{
		Time:      time.Now().Format("15:04:05.000"),
		Direction: dir,
		Type:      fmt.Sprintf("%T", pk),
		Data:      pk,
	}
	_ = p.encoder.Encode(rec)
}

func (p *PacketRecorder) HandlePacket(ctx *Context) {
	switch ctx.Direction {
	case FromClient:
		p.record("client", ctx.Packet)
	case FromServer:
		p.record("server", ctx.Packet)
	}
}
