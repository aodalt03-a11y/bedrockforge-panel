package mods

import (
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type PlayerState struct {
	mu        sync.RWMutex
	position  [3]float32
	Yaw       float32
	Pitch     float32
	HeldSlot  byte
	HeldItem  protocol.ItemInstance
	Health    float32
	Hunger    float32
	GameMode  int32
	Dimension int32
	RuntimeID uint64
	Inventory map[byte]protocol.ItemInstance
}

var State = &PlayerState{
	Inventory: make(map[byte]protocol.ItemInstance),
}

func (s *PlayerState) Pos() (float32, float32, float32) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.position[0], s.position[1], s.position[2]
}

func (s *PlayerState) YawPitch() (float32, float32) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Yaw, s.Pitch
}

type StateTracker struct{}

func init() { Register(&StateTracker{}) }

func (s *StateTracker) Name() string        { return "StateTracker" }
func (s *StateTracker) Init(_ *Context, _ any) {}
func (s *StateTracker) Tick()               {}

func (s *StateTracker) HandlePacket(ctx *Context) {
	switch pk := ctx.Packet.(type) {
	case *packet.MovePlayer:
		if ctx.Direction == FromServer && pk.EntityRuntimeID == State.RuntimeID {
			State.mu.Lock()
			State.position = [3]float32{pk.Position[0], pk.Position[1], pk.Position[2]}
			State.Yaw = pk.Yaw
			State.Pitch = pk.Pitch
			State.mu.Unlock()
		}
	case *packet.StartGame:
		if ctx.Direction == FromServer {
			State.mu.Lock()
			State.RuntimeID = pk.EntityRuntimeID
			State.GameMode = pk.PlayerGameMode
			State.position = [3]float32{pk.PlayerPosition[0], pk.PlayerPosition[1], pk.PlayerPosition[2]}
			State.mu.Unlock()
		}
	case *packet.UpdateAttributes:
		if ctx.Direction == FromServer {
			State.mu.Lock()
			for _, attr := range pk.Attributes {
				switch attr.Name {
				case "minecraft:health":
					State.Health = attr.Value
				case "minecraft:player.hunger":
					State.Hunger = attr.Value
				}
			}
			State.mu.Unlock()
		}
	case *packet.MobEquipment:
		if ctx.Direction == FromServer && pk.EntityRuntimeID == State.RuntimeID {
			State.mu.Lock()
			State.HeldSlot = pk.InventorySlot
			State.HeldItem = pk.NewItem
			State.mu.Unlock()
		}
	case *packet.InventorySlot:
		if ctx.Direction == FromServer {
			State.mu.Lock()
			State.Inventory[byte(pk.Slot)] = pk.NewItem
			State.mu.Unlock()
		}
	case *packet.InventoryContent:
		if ctx.Direction == FromServer {
			State.mu.Lock()
			for i, item := range pk.Content {
				State.Inventory[byte(i)] = item
			}
			State.mu.Unlock()
		}
	case *packet.Respawn:
		if ctx.Direction == FromServer {
			State.mu.Lock()
			State.position = [3]float32{pk.Position[0], pk.Position[1], pk.Position[2]}
			State.mu.Unlock()
		}
	}
}
