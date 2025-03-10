package galaxy

import "github.com/google/uuid"

// Player represents a unique player in a game.
type Player struct {
	PlayerID uuid.UUID
	Position Vector2D
	Radius   uint32

	// The skin the player currently is using,
	// implemented for now as a simple RGB color.
	Skin uint32
}
