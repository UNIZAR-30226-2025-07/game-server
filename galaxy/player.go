package galaxy

import (
	"log"
	"math/rand"
	"sync"
	"time"

	pb "galaxy.io/server/proto"
	"github.com/google/uuid"
)

const (
	STARTING_RADIUS = 50
)

type Log struct {
	sync.Mutex
	// Puntuación obtenida
	Score uint32
	// Jugadores eliminados
	KilledPlayers uint32
	// Segundos jugados
	TimeStart time.Time
	TimeEnd time.Time
}

// Player represents a unique player in a game.
type Player struct {
	sync.RWMutex
	PlayerID uuid.UUID
	ConnectionID uuid.UUID
	Position *Vector2D
	Radius   uint32
	Username string
	Stats Log

	// The skin the player currently is using,
	// implemented for now as a simple RGB color.
	Color uint32

	Skin *string

	conn ClientConnection
}


func NewPlayer(connectionID uuid.UUID, conn ClientConnection) *Player {
	return &Player{
		// PlayerID: playerID,
		ConnectionID: connectionID,
		Position: randomPosition(),
		Radius: STARTING_RADIUS,
		Color: FoodColors[rand.Intn(len(FoodColors))],
		Skin: nil,
		conn: conn,
		Username: "UNKNOWN",
		Stats: Log{},
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

func (p *Player) UpdatePlayerID(playerID uuid.UUID) {
	p.PlayerID = playerID;
}

func (p *Player) UpdateUsername(username string) {
	p.Username = username;
}

func (p *Player) UpdateColor(color uint32) {
	p.Color = color;
}

func (p *Player) UpdateSkin(skin string) {
	p.Skin = &skin;
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
	log.Printf("updating player radius, player = %v, old = %v, new = %v", p.PlayerID, p.Radius, radius);
	p.Lock();
	p.Radius = radius;

	p.Stats.Lock();
	p.Stats.Score = radius;
	p.Stats.Unlock();

	p.Unlock();
}
