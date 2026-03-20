package mods

import (
	"encoding/json"
	"fmt"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type Welcome struct {
	sendToClient func(pk packet.Packet) error
}

func init() { Register(&Welcome{}) }
func (w *Welcome) Name() string { return "Welcome" }
func (w *Welcome) Tick()        {}

func (w *Welcome) Init(ctx *Context, _ any) {
	w.sendToClient = ctx.SendToClient
}

func (w *Welcome) HandlePacket(ctx *Context) {
	if ctx.Direction == FromClient {
		if pk, ok := ctx.Packet.(*packet.Text); ok && pk.Message == "%help" {
			w.handleHelp()
			ctx.Drop = true
			return
		}
	}
	if ctx.Direction != FromServer {
		return
	}
	if _, ok := ctx.Packet.(*packet.StartGame); !ok {
		return
	}
	w.printCmds()
	w.send("§8§l----------------------------")
	w.send("§a§lBedrockForge §f§lv2 Active")
	w.send("§8§l----------------------------")
	w.send("§etype §e%help §ffor commands")
	w.send("§8§l----------------------------")
}

func (w *Welcome) send(msg string) {
	if w.sendToClient == nil {
		return
	}
	_ = w.sendToClient(&packet.Text{
		TextType: packet.TextTypeSystem,
		Message:  msg,
	})
}

func (w *Welcome) handleHelp() {
	w.send("§8§l================================")
	w.send("§a§lBedrockForge v2 Commands")
	w.send("§8§l================================")
	w.send("§e%help §f- this menu")
	w.send("§e%mods §f- list all modules")
	w.send("§e%mod <name> §f- toggle module")
	w.send("§e%set <mod> <key> <val> §f- change setting")
	w.send("§e%info <mod> §f- module info + settings")
	w.send("§8§l================================")
	w.send("§a§lSchematics")
	w.send("§e%schem list §f- list schematics")
	w.send("§e%schem load <n> §f- load schematic")
	w.send("§e%schem origin §f- anchor at feet")
	w.send("§e%schem layer <n|up|down> §f- set layer")
	w.send("§e%schem next/reset/status/materials/ghost/clear")
	w.send("§e%printer start/stop/delay <n>")
	w.send("§8§l================================")
}

func (w *Welcome) printCmds() {
	type CmdEntry struct {
		Command     string `json:"cmd"`
		Description string `json:"desc"`
		Script      string `json:"script"`
	}
	var entries []CmdEntry
	for _, m := range All() {
		if mod, ok := m.(*Module); ok {
			entries = append(entries, CmdEntry{
				Command:     "%mod " + mod.name,
				Description: mod.category + ": " + mod.desc,
				Script:      "modules",
			})
		}
	}
	if data, err := json.Marshal(entries); err == nil {
		fmt.Println("[CMDS]" + string(data))
	}
}

// SendWelcome is called after mods are initialized to send the welcome message
func SendWelcome(sendToClient func(packet.Packet) error) {
	send := func(msg string) {
		_ = sendToClient(&packet.Text{TextType: packet.TextTypeSystem, Message: msg})
	}
	send("\xa78\xa7l----------------------------")
	send("\xa7a\xa7lBedrockForge \xa7f\xa7lv2 Active")
	send("\xa78\xa7l----------------------------")
	send("\xa7etype \xa7e%help \xa7ffor commands")
	send("\xa78\xa7l----------------------------")
}
