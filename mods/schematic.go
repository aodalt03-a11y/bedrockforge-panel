package mods

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ── Types ─────────────────────────────────────────────────────────────────────

type SchematicBlock struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Z     int    `json:"z"`
	Block string `json:"block"`
}

type Schematic struct {
	Blocks []SchematicBlock `json:"blocks"`
}

// ── Mod ───────────────────────────────────────────────────────────────────────

type SchemMod struct {
	sendToClient func(pk packet.Packet) error
	sendToServer func(pk packet.Packet) error

	loaded      *Schematic
	name        string
	origin      [3]int
	originSet   bool
	currentIdx  int
	currentLayer int
	ghostOn     bool
	ghostSent   map[[3]int]bool

	// Printer
	printing    bool
	printDelay  int
	printTick   int
}

func init() { Register(&SchemMod{ghostSent: make(map[[3]int]bool), ghostOn: true, printDelay: 5}) }

func (s *SchemMod) Name() string { return "Schematic" }
func (s *SchemMod) Init(ctx *Context, _ any) {
	s.sendToClient = ctx.SendToClient
	s.sendToServer = ctx.SendToServer
}

func (s *SchemMod) chat(msg string) {
	if s.sendToClient == nil { return }
	_ = s.sendToClient(&packet.Text{
		TextType: packet.TextTypeSystem,
		Message:  "\xa78[\xa7aSchem\xa78] \xa7r" + msg,
	})
}

// ── Tick ──────────────────────────────────────────────────────────────────────

func (s *SchemMod) Tick() {
	if s.loaded == nil || !s.originSet { return }

	// Ghost blocks — refresh nearby every 2 ticks
	if s.ghostOn {
		s.refreshGhosts()
	}

	// Printer auto-place
	if s.printing {
		s.printTick++
		if s.printTick >= s.printDelay {
			s.printTick = 0
			s.placeNext()
		}
	}
}

func (s *SchemMod) refreshGhosts() {
	px, py, pz := State.Pos()
	const radius = 48
	sent := 0
	for i := s.currentIdx; i < len(s.loaded.Blocks) && sent < 50; i++ {
		b := s.loaded.Blocks[i]
		if b.Y != s.currentLayer { continue }
		wx := s.origin[0] + b.X
		wy := s.origin[1] + b.Y
		wz := s.origin[2] + b.Z
		dx := float64(wx) - float64(px)
		dy := float64(wy) - float64(py)
		dz := float64(wz) - float64(pz)
		if math.Sqrt(dx*dx+dy*dy+dz*dz) > radius { continue }
		key := [3]int{wx, wy, wz}
		if s.ghostSent[key] { continue }
		s.ghostSent[key] = true
		rid := RuntimeIDForBlock(b.Block)
		if rid == 0 { rid = RuntimeIDForBlock("glass") }
		s.setBlock(wx, wy, wz, rid)
		sent++
	}
}

func (s *SchemMod) clearGhosts() {
	for key := range s.ghostSent {
		s.setBlock(key[0], key[1], key[2], 0)
	}
	s.ghostSent = make(map[[3]int]bool)
}

func (s *SchemMod) setBlock(x, y, z int, rid uint32) {
	if s.sendToClient == nil { return }
	_ = s.sendToClient(&packet.UpdateBlock{
		Position:        [3]int32{int32(x), int32(y), int32(z)},
		NewBlockRuntimeID: rid,
		Flags:           packet.BlockUpdateNetwork,
		Layer:           0,
	})
}

func (s *SchemMod) placeNext() {
	if s.currentIdx >= len(s.loaded.Blocks) {
		s.printing = false
		s.chat("\xa7aPrinter done!")
		return
	}
	b := s.loaded.Blocks[s.currentIdx]
	for b.Y != s.currentLayer && s.currentIdx < len(s.loaded.Blocks) {
		s.currentIdx++
		if s.currentIdx >= len(s.loaded.Blocks) { return }
		b = s.loaded.Blocks[s.currentIdx]
	}
	wx := int32(s.origin[0] + b.X)
	wy := int32(s.origin[1] + b.Y)
	wz := int32(s.origin[2] + b.Z)
	rid := RuntimeIDForBlock(b.Block)
	if s.sendToServer != nil {
		_ = s.sendToServer(&packet.UpdateBlock{
			Position:          [3]int32{wx, wy, wz},
			NewBlockRuntimeID: rid,
			Flags:             packet.BlockUpdateNetwork,
			Layer:             0,
		})
	}
	// Clear ghost
	key := [3]int{int(wx), int(wy), int(wz)}
	delete(s.ghostSent, key)
	s.currentIdx++
}

func (s *SchemMod) printStatus() {
	if s.loaded == nil { s.chat("No schematic loaded"); return }
	pct := 0
	if len(s.loaded.Blocks) > 0 {
		pct = s.currentIdx * 100 / len(s.loaded.Blocks)
	}
	s.chat(fmt.Sprintf("\xa7f%s \xa77| \xa7e%d/%d \xa77(\xa7a%d%%\xa77) | layer \xa7b%d",
		s.name, s.currentIdx, len(s.loaded.Blocks), pct, s.currentLayer))
}

func (s *SchemMod) printMaterials() {
	if s.loaded == nil { s.chat("No schematic loaded"); return }
	counts := map[string]int{}
	for _, b := range s.loaded.Blocks {
		counts[b.Block]++
	}
	s.chat("Materials:")
	for name, count := range counts {
		stacks := count / 64
		rem := count % 64
		var str string
		if stacks > 0 { str = fmt.Sprintf("%dx64", stacks) }
		if rem > 0 {
			if str != "" { str += "+" }
			str += strconv.Itoa(rem)
		}
		s.chat(fmt.Sprintf("  \xa7f%s\xa77: %s", name, str))
	}
}

func (s *SchemMod) listSchematics() {
	dir := getSchemDir()
	entries, err := os.ReadDir(dir)
	if err != nil { s.chat("No schematics folder found"); return }
	var names []string
	for _, e := range entries {
		ext := filepath.Ext(e.Name())
		if ext == ".json" || ext == ".litematic" || ext == ".schem" {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 { s.chat("No schematics found"); return }
	s.chat(fmt.Sprintf("\xa7e%d schematic(s):", len(names)))
	for _, n := range names { s.chat("  \xa7f" + n) }
}

func getSchemDir() string {
	if d := os.Getenv("BF_SCHEMATICS_DIR"); d != "" { return d }
	return "schematics"
}

func (s *SchemMod) loadSchem(name string) {
	dir := getSchemDir()
	var path string
	for _, candidate := range []string{
		filepath.Join(dir, name),
		filepath.Join(dir, name+".json"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			path = candidate
			break
		}
	}
	if path == "" { s.chat("Not found: " + name); return }
	data, err := os.ReadFile(path)
	if err != nil { s.chat("Read error: " + err.Error()); return }
	var sc Schematic
	if err := json.Unmarshal(data, &sc); err != nil { s.chat("Bad JSON: " + err.Error()); return }
	s.clearGhosts()
	s.loaded = &sc
	s.name = name
	s.originSet = false
	s.currentIdx = 0
	s.currentLayer = 0
	s.printing = false
	s.chat(fmt.Sprintf("\xa7aLoaded \xa7f%s \xa77(\xa7e%d\xa77 blocks)", name, len(sc.Blocks)))
	s.chat("Type \xa7e%schem origin\xa7r to anchor at your feet")
}

// ── Packet handler ────────────────────────────────────────────────────────────

func (s *SchemMod) HandlePacket(ctx *Context) {
	if ctx.Direction != FromClient { return }
	var msg string
	switch pk := ctx.Packet.(type) {
	case *packet.Text:
		msg = strings.TrimSpace(pk.Message)
	case *packet.CommandRequest:
		msg = strings.TrimSpace(pk.CommandLine)
		if strings.HasPrefix(msg, "/") { msg = msg[1:] }
	default:
		return
	}
	parts := strings.Fields(msg)
	if len(parts) == 0 || parts[0] != "%schem" && parts[0] != "%printer" { return }

	ctx.Drop = true

	if parts[0] == "%printer" {
		if len(parts) < 2 { s.chat("Usage: %printer start/stop/delay <n>"); return }
		switch parts[1] {
		case "start":
			if s.loaded == nil { s.chat("Load a schematic first"); return }
			if !s.originSet { s.chat("Set origin first"); return }
			s.printing = true
			s.chat("\xa7aPrinter started")
		case "stop":
			s.printing = false
			s.chat("\xa7cPrinter stopped")
		case "delay":
			if len(parts) >= 3 {
				n, _ := strconv.Atoi(parts[2])
				if n > 0 { s.printDelay = n }
				s.chat(fmt.Sprintf("Delay: \xa7e%d ticks", s.printDelay))
			}
		}
		return
	}

	if len(parts) < 2 {
		s.chat("Commands: list, load <n>, origin, layer <n|up|down>, next, reset, status, materials, ghost, clear")
		return
	}

	switch parts[1] {
	case "list":
		s.listSchematics()
	case "load":
		if len(parts) < 3 { s.chat("Usage: %schem load <name>"); return }
		s.loadSchem(parts[2])
	case "origin":
		if s.loaded == nil { s.chat("Load a schematic first"); return }
		px, py, pz := State.Pos()
		if len(parts) >= 5 {
			x, _ := strconv.Atoi(parts[2])
			y, _ := strconv.Atoi(parts[3])
			z, _ := strconv.Atoi(parts[4])
			s.origin = [3]int{x, y, z}
		} else {
			s.origin = [3]int{int(px), int(py), int(pz)}
		}
		s.originSet = true
		s.currentIdx = 0
		s.ghostSent = make(map[[3]int]bool)
		s.chat(fmt.Sprintf("\xa7aOrigin set: \xa7f(%d, %d, %d) \xa77| \xa7e%d blocks",
			s.origin[0], s.origin[1], s.origin[2], len(s.loaded.Blocks)))
	case "layer":
		if s.loaded == nil { s.chat("Load a schematic first"); return }
		if len(parts) < 3 { s.chat("Usage: %schem layer <n|up|down>"); return }
		s.clearGhosts()
		switch parts[2] {
		case "up":   s.currentLayer++
		case "down": s.currentLayer--
		default:
			n, _ := strconv.Atoi(parts[2])
			s.currentLayer = n
		}
		s.currentIdx = 0
		s.chat(fmt.Sprintf("Layer: \xa7e%d", s.currentLayer))
	case "next":
		if !s.originSet { s.chat("Set origin first"); return }
		if s.currentIdx < len(s.loaded.Blocks) {
			b := s.loaded.Blocks[s.currentIdx]
			key := [3]int{s.origin[0]+b.X, s.origin[1]+b.Y, s.origin[2]+b.Z}
			delete(s.ghostSent, key)
			s.setBlock(key[0], key[1], key[2], 0)
			s.currentIdx++
		}
		// Print next block info
		for s.currentIdx < len(s.loaded.Blocks) {
			b := s.loaded.Blocks[s.currentIdx]
			if b.Y == s.currentLayer {
				wx := s.origin[0]+b.X; wy := s.origin[1]+b.Y; wz := s.origin[2]+b.Z
				px, py, pz := State.Pos()
				dx := float64(wx)-float64(px); dy := float64(wy)-float64(py); dz := float64(wz)-float64(pz)
				dist := math.Sqrt(dx*dx+dy*dy+dz*dz)
				s.chat(fmt.Sprintf("[\xa7e%d/%d\xa77] \xa7f%s \xa77(\xa7b%d,%d,%d\xa77) dist \xa7e%.1f",
					s.currentIdx, len(s.loaded.Blocks), b.Block, wx, wy, wz, dist))
				return
			}
			s.currentIdx++
		}
		s.chat(fmt.Sprintf("\xa7aLayer %d complete! Use %%schem layer up", s.currentLayer))
	case "reset":
		if s.loaded == nil { s.chat("Load a schematic first"); return }
		s.clearGhosts()
		s.currentIdx = 0
		s.currentLayer = 0
		s.printing = false
		s.chat("\xa7aReset to block 1")
	case "status":
		s.printStatus()
	case "materials":
		s.printMaterials()
	case "ghost":
		s.ghostOn = !s.ghostOn
		if s.ghostOn {
			s.chat("\xa7aGhost blocks ON")
		} else {
			s.clearGhosts()
			s.chat("\xa7cGhost blocks OFF")
		}
	case "clear":
		s.clearGhosts()
		s.loaded = nil
		s.name = ""
		s.originSet = false
		s.printing = false
		s.chat("\xa7aCleared")
	}
}
