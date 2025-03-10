package events

import (
	"galaxy.io/server/galaxy/utils"
	"github.com/google/uuid"
)

type PlayerMove struct {
  PlayerID uuid.UUID
  Position utils.Vector2D
}
