package operations

import "galaxy.io/server/galaxy"

type Operation interface {
  Process(world *galaxy.World)
}

type OperationType uint16

const (
  OpUnused = iota
  OpJoin
  OpLeave
  OpMove
  OpEatPlayer
  OpEatFood
)
