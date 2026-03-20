package mods

import (
"encoding/json"
"fmt"
"log"
"os"
"strings"
"sync"

"github.com/sandertv/gophertunnel/minecraft/protocol"
"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type BlockRegistry struct {
mu       sync.RWMutex
nameToID map[string]uint32
}

var globalBlockRegistry = &BlockRegistry{
nameToID: make(map[string]uint32),
}

func init() {
Register(globalBlockRegistry)
loadStaticPalette()
}

func loadStaticPalette() {
// Try Geyser extracted palette first
files := []string{"bedrock_blocks.json", "block_ids.json"}
for _, f := range files {
data, err := os.ReadFile(f)
if err != nil {
continue
}
var raw map[string]uint32
if err := json.Unmarshal(data, &raw); err != nil {
// Try old format with runtime_id field
var raw2 map[string]struct {
RuntimeID int `json:"runtime_id"`
}
if err2 := json.Unmarshal(data, &raw2); err2 != nil {
continue
}
globalBlockRegistry.mu.Lock()
for name, entry := range raw2 {
clean := stripNamespace(name)
globalBlockRegistry.nameToID[clean] = uint32(entry.RuntimeID)
globalBlockRegistry.nameToID[name] = uint32(entry.RuntimeID)
}
globalBlockRegistry.mu.Unlock()
log.Printf("[BlockRegistry] loaded %d blocks from %s", len(raw2), f)
return
}
globalBlockRegistry.mu.Lock()
for name, id := range raw {
clean := stripNamespace(name)
globalBlockRegistry.nameToID[clean] = id
globalBlockRegistry.nameToID[name] = id
}
globalBlockRegistry.mu.Unlock()
log.Printf("[BlockRegistry] loaded %d blocks from %s", len(raw), f)
return
}
log.Printf("[BlockRegistry] no palette file found")
}

func (b *BlockRegistry) Name() string          { return "BlockRegistry" }
func (b *BlockRegistry) Init(_ *Context, _ any) {}
func (b *BlockRegistry) Tick()                 {}

func (b *BlockRegistry) HandlePacket(ctx *Context) {
if ctx.Direction != FromServer {
return
}
pk, ok := ctx.Packet.(*packet.StartGame)
if !ok || len(pk.Blocks) == 0 {
return
}
b.mu.Lock()
defer b.mu.Unlock()
for i, entry := range pk.Blocks {
clean := stripNamespace(entry.Name)
b.nameToID[clean] = uint32(i)
b.nameToID[entry.Name] = uint32(i)
}
log.Printf("[BlockRegistry] updated %d blocks from server palette", len(pk.Blocks))
f, err := os.Create("palette_dump.txt")
if err == nil {
defer f.Close()
for name, id := range b.nameToID {
fmt.Fprintf(f, "%d\t%s\n", id, name)
}
}
}

func RuntimeIDForBlock(name string) uint32 {
globalBlockRegistry.mu.RLock()
defer globalBlockRegistry.mu.RUnlock()
if id, ok := globalBlockRegistry.nameToID[name]; ok {
return id
}
clean := stripNamespace(name)
if id, ok := globalBlockRegistry.nameToID[clean]; ok {
return id
}
return 0
}

func stripNamespace(s string) string {
if idx := strings.Index(s, ":"); idx >= 0 {
return s[idx+1:]
}
return s
}

var _ = protocol.BlockEntry{}
