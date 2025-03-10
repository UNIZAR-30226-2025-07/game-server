package galaxy

import (
	"sync"

	"github.com/google/uuid"
)


// PlayerID is a UUID v4 identifying a unique player.
// This identifier will be shared with the database.
type PlayerID uuid.UUID

// World holds all elements inside a current game, this includes players, bots and food.
// World is locked behind a mutex in order to archieve safe concurrency.
// Each server should only contain one world at the moment.
type World struct {
  sync.RWMutex
  Players map[uuid.UUID]Player
  Food []Food
}
