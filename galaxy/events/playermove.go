package events

import (
	"galaxy.io/server/galaxy"
	"github.com/google/uuid"
)

type PlayerMove struct {
  PlayerID uuid.UUID
  Position galaxy.Vector2D
}
