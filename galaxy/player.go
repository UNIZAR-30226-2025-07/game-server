package galaxy

import (
	"sync"

	"github.com/google/uuid"
	pb "galaxy.io/server/proto"
)

const (
	STARTING_RADIUS = 50
)

// Player represents a unique player in a game.
type Player struct {
	sync.RWMutex
	PlayerID uuid.UUID
	Position *Vector2D
	Radius   uint32

	// The skin the player currently is using,
	// implemented for now as a simple RGB color.
	Skin uint32

	conn ClientConnection
}


func NewPlayer(playerID uuid.UUID, conn ClientConnection) *Player {
	return &Player{
		PlayerID: playerID,
		Position: randomPosition(),
		Radius: STARTING_RADIUS,
		Skin: 0,
		conn: conn,
	}
}

func (p *Player) SendEvent(event *pb.Event) error {
	return p.conn.SendEvent(event)
}

func (p *Player) Disconnect() {
	if p.conn != nil {
		p.conn.Close()
	}
}

func (p *Player) UpdatePosition(position *Vector2D) {
	p.Lock()
	p.Position = position
	p.Unlock()
}

func (p *Player) GetPosition() *Vector2D {
	p.RLock()
	defer p.RUnlock()
	return &Vector2D{
		X: p.Position.X,
		Y: p.Position.Y,
	}
}

func (p *Player) UpdateRadius(radius uint32) {
	p.Lock()
	p.Radius = radius
	p.Unlock()
}
