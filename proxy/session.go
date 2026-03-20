package proxy

import (
	"log/slog"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"

	"bedrockforge/mods"
)

var logger = slog.Default()

// runTicker starts the 50ms mod tick loop, returns a stop func
func runTicker(allMods []mods.Mod) func() {
	stop := make(chan struct{})
	go func() {
		t := time.NewTicker(50 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				for _, m := range allMods {
					m.Tick()
				}
			case <-stop:
				return
			}
		}
	}()
	return func() { close(stop) }
}

// initMods initialises all mods with the given send functions
func initMods(allMods []mods.Mod, sendToClient, sendToServer func(packet.Packet) error) *mods.Context {
	base := &mods.Context{
		SendToClient: sendToClient,
		SendToServer: sendToServer,
	}
	for _, m := range allMods {
		m.Init(base, nil)
	}
	return base
}

// pipe reads packets from src, runs them through mods, writes to dst
func pipe(src, dst *minecraft.Conn, dir mods.Direction, allMods []mods.Mod, base *mods.Context) error {
	for {
		pk, err := src.ReadPacket()
		if err != nil {
			return err
		}
		ctx := &mods.Context{
			Packet:       pk,
			Direction:    dir,
			SendToClient: base.SendToClient,
			SendToServer: base.SendToServer,
		}
		for _, m := range allMods {
			m.HandlePacket(ctx)
			if ctx.Drop {
				break
			}
		}
		if !ctx.Drop {
			if err := dst.WritePacket(ctx.Packet); err != nil {
				return err
			}
		}
	}
}
