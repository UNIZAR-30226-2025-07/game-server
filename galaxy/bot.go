package galaxy

import (
	"math"
	"time"

	"galaxy.io/server/proto"
)

type Bot struct {
	player *Player
}

func (b *Bot) Start(w *World) {
	for {
		// check colisions
		b.checkColision(w)

		// pathfind
		b.performPathfinding(w)

		time.Sleep(20*time.Millisecond)
	}
}

func (b *Bot) surface() uint32 {
	x := float64(b.player.Position.X)
	y := float64(b.player.Position.Y)
	sf := math.Sqrt(x*x + y*y)
	return uint32(math.Floor(sf))
}

func (b *Bot) checkColision(w *World) {
	// check colisions with food
	surface := b.surface()
	w.foodMutex.RLock()
	for _, food := range w.food {
		if distance(b.player.Position, &food.position) < surface {
			w.operationPlayerEatFood(b.player, &proto.EatFoodOperation{
				FoodPosition: food.position.toPacket(),
			})
			return
		}
	}
	w.foodMutex.RUnlock()

	w.playersMutex.RLock()
	for _, player := range w.players {
		if distance(b.player.Position, player.Position) < surface {
			newRadius := b.player.Radius + player.Radius
			w.operationEatPlayer(b.player, &proto.EatPlayerOperation{
				PlayerEaten: player.PlayerID[:],
				NewRadius: &newRadius,
			})
			return
		}
	}
	w.playersMutex.RUnlock()
}


func (b *Bot) performPathfinding(w *World) {
	var targets []*Vector2D

	w.foodMutex.RLock()
	for _, food := range w.food {
		dist := distance(b.player.Position, &food.position)
		if dist < MAX_RANGE {
			targets = append(targets, &food.position)
		}
	}
	w.foodMutex.RUnlock()

	w.playersMutex.RLock()
	for _, player := range w.players {
		dist := distance(b.player.Position, player.Position)
		if dist < MAX_RANGE {
			targets = append(targets, player.Position)
		}
	}
	w.playersMutex.RUnlock()

	if len(targets) == 0 {
		return
	}

	bestTarget := targets[0]
	for _, target := range targets {
		if (distance(b.player.Position, target) < distance(b.player.Position, bestTarget)) {
			bestTarget = target
		}
	}

	b.moveTowards(w, bestTarget);
}

func (b *Bot) moveTowards(w *World, target *Vector2D) {
	var newX uint32
	var newY uint32

	deltaX := b.player.Position.X - target.X
	if deltaX > SPEED {
		newX = b.player.Position.X - SPEED
	} else {
		newX = deltaX
	}

	deltaY := b.player.Position.Y - target.Y
	if deltaY > SPEED {
		newY = b.player.Position.Y - SPEED
	} else {
		newY = deltaY
	}

	newPosition := Vector2D{
		X: newX,
		Y: newY,
	}

	w.operationPlayerMove(b.player, &proto.MoveOperation{
		Position: newPosition.toPacket(),
	})
}


func distance(v1 *Vector2D, v2 *Vector2D) uint32 {
	dx := v1.X - v2.X
	dy := v1.Y - v2.Y
	dist := math.Floor(math.Sqrt(float64(dx*dx + dy*dy)))
	return uint32(dist)
}

const (
	MAX_RANGE = 2000
	SPEED = 80
)
