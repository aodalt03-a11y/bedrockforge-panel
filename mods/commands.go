package mods

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type CommandRouter struct {
	sendToClient func(pk packet.Packet) error
}

func init() { Register(&CommandRouter{}) }
func (c *CommandRouter) Name() string { return "CommandRouter" }
func (c *CommandRouter) Tick()        {}

func (c *CommandRouter) Init(ctx *Context, _ any) {
	c.sendToClient = ctx.SendToClient
}

func (c *CommandRouter) HandlePacket(ctx *Context) {
	if ctx.Direction != FromClient {
		return
	}
	var msg string
	switch pk := ctx.Packet.(type) {
	case *packet.Text:
		msg = strings.TrimSpace(pk.Message)
	case *packet.CommandRequest:
		msg = strings.TrimSpace(pk.CommandLine)
		// strip leading / if present
		if strings.HasPrefix(msg, "/") {
			msg = msg[1:]
		}
	default:
		return
	}
	if !strings.HasPrefix(msg, "%") {
		return
	}
	parts := strings.Fields(msg)
	if len(parts) == 0 {
		return
	}
	switch parts[0] {
	case "%mods":
		c.handleMods()
		ctx.Drop = true
	case "%mod":
		if len(parts) >= 2 {
			c.handleToggle(strings.Join(parts[1:], " "))
			ctx.Drop = true
		}
	case "%set":
		if len(parts) >= 4 {
			c.handleSet(parts[1], parts[2], strings.Join(parts[3:], " "))
			ctx.Drop = true
		}
	case "%info":
		if len(parts) >= 2 {
			c.handleInfo(strings.Join(parts[1:], " "))
			ctx.Drop = true
		}
	// %schem and %printer are handled by schematic.go - do NOT drop here
	}
}

func (c *CommandRouter) send(msg string) {
	if c.sendToClient == nil {
		return
	}
	_ = c.sendToClient(&packet.Text{
		TextType: packet.TextTypeSystem,
		Message:  msg,
	})
}

func (c *CommandRouter) handleMods() {
	cats := map[string][]*Module{}
	for _, m := range All() {
		mod, ok := m.(*Module)
		if !ok {
			continue
		}
		cats[mod.category] = append(cats[mod.category], mod)
	}
	c.send("§b=== BedrockForge Modules ===")
	for cat, list := range cats {
		c.send("§e" + cat + ":")
		for _, mod := range list {
			state := "§cOFF"
			if mod.enabled {
				state = "§aON"
			}
			c.send("  §f" + mod.name + " " + state)
		}
	}
}

func (c *CommandRouter) handleToggle(name string) {
	for _, m := range All() {
		mod, ok := m.(*Module)
		if !ok {
			continue
		}
		if strings.EqualFold(mod.name, name) {
			mod.enabled = !mod.enabled
			state := "§cOFF"
			if mod.enabled {
				state = "§aON"
				if mod.onEnable != nil {
					mod.onEnable()
				}
			} else {
				if mod.onDisable != nil {
					mod.onDisable()
				}
			}
			c.send(fmt.Sprintf("§e[BF] §f%s %s", mod.name, state))
			return
		}
	}
	c.send("§c[BF] Unknown module: " + name)
}

func (c *CommandRouter) handleSet(modName, key, val string) {
	for _, m := range All() {
		mod, ok := m.(*Module)
		if !ok {
			continue
		}
		if !strings.EqualFold(mod.name, modName) {
			continue
		}
		s, ok := mod.settings[key]
		if !ok {
			c.send("§c[BF] Unknown setting: " + key)
			return
		}
		switch s.typ {
		case "float":
			n, err := strconv.ParseFloat(val, 64)
			if err != nil {
				c.send("§c[BF] Invalid value")
				return
			}
			if s.min != 0 || s.max != 0 {
				if n < s.min { n = s.min }
				if n > s.max { n = s.max }
			}
			s.fval = n
		case "int":
			n, err := strconv.ParseFloat(val, 64)
			if err != nil {
				c.send("§c[BF] Invalid value")
				return
			}
			if s.min != 0 || s.max != 0 {
				if n < s.min { n = s.min }
				if n > s.max { n = s.max }
			}
			s.fval = float64(int(n))
		case "bool":
			s.bval = val == "true" || val == "1" || val == "on"
		case "string", "enum":
			s.sval = val
		}
		c.send(fmt.Sprintf("§a[BF] %s.%s = %s", mod.name, key, val))
		return
	}
	c.send("§c[BF] Unknown module: " + modName)
}

func (c *CommandRouter) handleInfo(name string) {
	for _, m := range All() {
		mod, ok := m.(*Module)
		if !ok {
			continue
		}
		if !strings.EqualFold(mod.name, name) {
			continue
		}
		c.send(fmt.Sprintf("§b[BF] %s §7(%s) - %s", mod.name, mod.category, mod.desc))
		for k, s := range mod.settings {
			var cur string
			switch s.typ {
			case "bool":
				cur = fmt.Sprintf("%v", s.bval)
			case "string", "enum":
				cur = s.sval
			default:
				cur = fmt.Sprintf("%v", s.fval)
			}
			c.send(fmt.Sprintf("  §e%s§7=§f%s §7(%s)", k, cur, s.desc))
		}
		return
	}
	c.send("§c[BF] Unknown module: " + name)
}
