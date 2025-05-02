package galaxy

import (
	"log"
	"sync"

	pb "galaxy.io/server/proto"
	"github.com/google/uuid"
)

const (
	STARTING_RADIUS = 50
)

type Log struct {
	// Puntuación obtenida
	Score uint32
	// Jugadores eliminadosk
	KilledPlayers uint32
	// Segundos jugados
	TimePlayed uint32
}

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
	err := p.conn.SendEvent(event)
	if err != nil {
		log.Printf("error in sendEvent: %v", err)
	}

	return err
}

func (p *Player) Disconnect() {
	log.Printf("disconnecting player %v", p.PlayerID)
	if p.conn != nil {
		p.conn.Close()
	}
}

func (p *Player) UpdatePosition(position *Vector2D) {
	// log.Printf("updating player position, player = %v, oldpos = %v, newpos = %v", p.PlayerID, p.Position, position)
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
	log.Printf("updating player radius, player = %v, old = %v, new = %v", p.PlayerID, p.Radius, radius)
	p.Lock()
	p.Radius = radius
	p.Unlock()
}
