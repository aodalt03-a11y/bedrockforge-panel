package mods

import (
"fmt"
"strings"

"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type Setting struct {
typ  string
desc string
fval float64
bval bool
sval string
min  float64
max  float64
}

func fset(desc string, def, min, max float64) *Setting {
return &Setting{typ: "float", desc: desc, fval: def, min: min, max: max}
}
func iset(desc string, def, min, max float64) *Setting {
return &Setting{typ: "int", desc: desc, fval: def, min: min, max: max}
}
func bset(desc string, def bool) *Setting {
return &Setting{typ: "bool", desc: desc, bval: def}
}
func eset(desc, def string) *Setting {
return &Setting{typ: "enum", desc: desc, sval: def}
}

type Module struct {
name         string
category     string
desc         string
enabled      bool
settings     map[string]*Setting
onEnable     func()
onDisable    func()
tickFn       func()
packetFn     func(ctx *Context)
sendToClient func(pk packet.Packet) error
sendToServer func(pk packet.Packet) error
tickCount    int64
}

func (m *Module) Name() string { return m.name }
func (m *Module) Tick() {
if !m.enabled || m.tickFn == nil {
return
}
m.tickCount++
m.tickFn()
}
func (m *Module) HandlePacket(ctx *Context) {
if m.packetFn == nil {
return
}
m.packetFn(ctx)
}
func (m *Module) Init(ctx *Context, _ any) {
m.sendToClient = ctx.SendToClient
m.sendToServer = ctx.SendToServer
}
func (m *Module) every(n int64) bool { return m.tickCount%n == 0 }
func (m *Module) f(key string) float64 {
if s, ok := m.settings[key]; ok {
return s.fval
}
return 0
}
func (m *Module) b(key string) bool {
if s, ok := m.settings[key]; ok {
return s.bval
}
return false
}
func (m *Module) s(key string) string {
if s, ok := m.settings[key]; ok {
return s.sval
}
return ""
}
func (m *Module) chat(msg string) {
if m.sendToClient == nil {
return
}
_ = m.sendToClient(&packet.Text{TextType: packet.TextTypeSystem, Message: msg})
}
func (m *Module) actionBar(msg string) {
if m.sendToClient == nil {
return
}
_ = m.sendToClient(&packet.SetTitle{ActionType: packet.TitleActionSetActionBar, Text: msg})
}
func (m *Module) movePlayer(x, y, z, yaw, pitch float32, onGround bool) {
if m.sendToServer == nil {
return
}
_ = m.sendToServer(&packet.MovePlayer{
EntityRuntimeID: State.RuntimeID,
Position:        [3]float32{x, y, z},
Pitch:           pitch,
Yaw:             yaw,
HeadYaw:         yaw,
Mode:            packet.MoveModeNormal,
OnGround:        onGround,
})
}

func newMod(name, category, desc string, settings map[string]*Setting) *Module {
if settings == nil {
settings = map[string]*Setting{}
}
m := &Module{name: name, category: category, desc: desc, settings: settings}
Register(m)
return m
}

func init() {
// DeathTracker - only utility kept
dt := newMod("DeathTracker", "Utility", "Track player deaths", map[string]*Setting{
"announce": bset("Print to terminal", true),
})
dt.packetFn = func(ctx *Context) {
if ctx.Direction != FromServer {
return
}
pk, ok := ctx.Packet.(*packet.Text)
if !ok {
return
}
msg := strings.ToLower(pk.Message)
if strings.Contains(msg, "was slain") || strings.Contains(msg, "died") || strings.Contains(msg, "was killed") {
if dt.b("announce") {
fmt.Println("[DeathTracker]", pk.Message)
}
}
}
_ = dt
}
