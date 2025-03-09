package operations

import (
	"galaxy.io/server/galaxy"
	"github.com/google/uuid"
)

type operationMoveRequest struct {
  // Identificador del jugador
  PlayerID  uuid.UUID
  // Nueva posicion
  Position  galaxy.Vector2D
  // Maybe checksum
  // checksum
}

func (op *operationMoveRequest) Process(world *galaxy.World) {
  panic("unimplemented")
}
