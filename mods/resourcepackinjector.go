package mods

import (
"fmt"
"os"
"path/filepath"
"strings"

"github.com/sandertv/gophertunnel/minecraft/resource"
)

type ResourcePackInjector struct {
Packs []*resource.Pack
}

func init() { Register(&ResourcePackInjector{}) }

func (r *ResourcePackInjector) Name() string { return "ResourcePackInjector" }

func (r *ResourcePackInjector) Init(ctx *Context, _ any) {
if len(r.Packs) == 0 {
r.loadPacks()
}
for _, p := range r.Packs {
fmt.Printf("[PackInjector] pack: %s %s\n", p.UUID(), p.Version())
}
fmt.Printf("[PackInjector] ready, %d packs\n", len(r.Packs))
}

func (r *ResourcePackInjector) Tick() {}

// LoadPacks can be called before Init to eagerly load packs for the listener
func LoadResourcePacks() []*resource.Pack {
r := &ResourcePackInjector{}
r.loadPacks()
return r.Packs
}

func getPacksDir() string {
if d := os.Getenv("BF_PACKS_DIR"); d != "" {
return d
}
return "packs"
}

func (r *ResourcePackInjector) loadPacks() {
packsDir := getPacksDir()
os.MkdirAll(packsDir, 0755)
entries, _ := os.ReadDir(packsDir)
for _, e := range entries {
if e.IsDir() {
continue
}
name := e.Name()
path := filepath.Join(packsDir, name)
if strings.HasSuffix(name, ".mcpack") || strings.HasSuffix(name, ".zip") {
pack, err := resource.ReadPath(path)
if err != nil {
fmt.Printf("[PackInjector] failed to read %s: %v\n", name, err)
continue
}
r.Packs = append(r.Packs, pack)
} else if strings.HasSuffix(name, ".url") {
content, err := os.ReadFile(path)
if err != nil {
continue
}
url := strings.TrimSpace(string(content))
pack, err := resource.ReadURL(url)
if err != nil {
continue
}
r.Packs = append(r.Packs, pack)
}
}
}
func (r *ResourcePackInjector) HandlePacket(ctx *Context) {}
