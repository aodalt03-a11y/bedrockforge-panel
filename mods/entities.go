package mods

import (
	"math"
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type EntityInfo struct {
	RuntimeID uint64
	Name      string
	Position  [3]float32
	Yaw       float32
	Pitch     float32
	IsPlayer  bool
	dist      float32
}

type EntityTracker struct {
	mu       sync.RWMutex
	entities map[uint64]*EntityInfo
}

var Entities = &EntityTracker{
	entities: make(map[uint64]*EntityInfo),
}

// Snapshot returns a copy of all tracked entities
func (e *EntityTracker) Snapshot() []EntityInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]EntityInfo, 0, len(e.entities))
	for _, ent := range e.entities {
		out = append(out, *ent)
	}
	return out
}

// Nearest returns the closest entity within range, optionally players only
func (e *EntityTracker) Nearest(playersOnly bool, maxRange float32) *EntityInfo {
	px, py, pz := State.Pos()
	e.mu.RLock()
	defer e.mu.RUnlock()
	var best *EntityInfo
	var bestDist float32 = maxRange + 1
	for _, ent := range e.entities {
		if ent.RuntimeID == State.RuntimeID { continue }
		if playersOnly && !ent.IsPlayer { continue }
		dx := ent.Position[0] - px
		dy := ent.Position[1] - py
		dz := ent.Position[2] - pz
		d := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
		if d < bestDist {
			bestDist = d
			copy := *ent
			copy.dist = d
			best = &copy
		}
	}
	return best
}

type EntityTrackerMod struct{}

func init() { Register(&EntityTrackerMod{}) }

func (e *EntityTrackerMod) Name() string        { return "EntityTracker" }
func (e *EntityTrackerMod) Init(_ *Context, _ any) {}
func (e *EntityTrackerMod) Tick()               {}

func (e *EntityTrackerMod) HandlePacket(ctx *Context) {
	if ctx.Direction != FromServer {
		return
	}
	switch pk := ctx.Packet.(type) {
	case *packet.AddPlayer:
		Entities.mu.Lock()
		Entities.entities[pk.EntityRuntimeID] = &EntityInfo{
			RuntimeID: pk.EntityRuntimeID,
			Name:      pk.Username,
			Position:  [3]float32{pk.Position[0], pk.Position[1], pk.Position[2]},
			Yaw:       pk.Yaw,
			Pitch:     pk.Pitch,
			IsPlayer:  true,
		}
		Entities.mu.Unlock()
	case *packet.AddActor:
		Entities.mu.Lock()
		Entities.entities[pk.EntityRuntimeID] = &EntityInfo{
			RuntimeID: pk.EntityRuntimeID,
			Name:      pk.EntityType,
			Position:  [3]float32{pk.Position[0], pk.Position[1], pk.Position[2]},
			Yaw:       pk.Yaw,
			Pitch:     pk.Pitch,
			IsPlayer:  false,
		}
		Entities.mu.Unlock()
	case *packet.MoveActorAbsolute:
		Entities.mu.Lock()
		if ent, ok := Entities.entities[pk.EntityRuntimeID]; ok {
			ent.Position = [3]float32{pk.Position[0], pk.Position[1], pk.Position[2]}
			ent.Yaw = pk.Rotation[0]
			ent.Pitch = pk.Rotation[1]
		}
		Entities.mu.Unlock()
	case *packet.MoveActorDelta:
		Entities.mu.Lock()
		if ent, ok := Entities.entities[pk.EntityRuntimeID]; ok {
			ent.Position = [3]float32{pk.Position[0], pk.Position[1], pk.Position[2]}
		}
		Entities.mu.Unlock()
	case *packet.MovePlayer:
		Entities.mu.Lock()
		if ent, ok := Entities.entities[pk.EntityRuntimeID]; ok {
			ent.Position = [3]float32{pk.Position[0], pk.Position[1], pk.Position[2]}
			ent.Yaw = pk.Yaw
			ent.Pitch = pk.Pitch
		}
		Entities.mu.Unlock()
	case *packet.RemoveActor:
		Entities.mu.Lock()
		delete(Entities.entities, uint64(pk.EntityUniqueID))
		Entities.mu.Unlock()
	}
}
