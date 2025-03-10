package operations

import (
	"galaxy.io/server/galaxy"
	"galaxy.io/server/galaxy/utils"
	"github.com/google/uuid"
)

type operationMoveRequest struct {
  // Identificador del jugador
  PlayerID  uuid.UUID
  // Nueva posicion
  Position  utils.Vector2D
  // Maybe checksum
  // checksum
}

func (op *operationMoveRequest) Process(world *galaxy.World) {
  panic("unimplemented")
}
