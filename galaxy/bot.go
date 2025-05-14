package galaxy

import (
	"log"
	"math"
	"math/rand"
	"time"

	"galaxy.io/server/proto"
	"github.com/google/uuid"
)

// GenerateConstellationName returns a random name based on constellations
func generateConstellationName() string {
	constellationNames := []string{
		"Andromeda",
		"Aquarius",
		"Aries",
		"Cassiopeia",
		"Cygnus",
		"Draco",
		"Gemini",
		"Lyra",
		"Orion",
		"Perseus",
		"Phoenix",
		"Pisces",
		"Sagittarius",
		"Scorpius",
		"Taurus",
		"Ursa",
		"Vela",
	}

	return constellationNames[rand.Intn(len(constellationNames))]
}

type Bot struct {
	player *Player
	target *Vector2D
	steps	 uint32
}

func NewBot() *Bot {
	player := NewPlayer(uuid.Nil, nil)

	uuid, _ := uuid.NewRandom()
	player.UpdatePlayerID(uuid)
	player.UpdateRadius(50)
	player.UpdateColor(randomColor())
	player.UpdateUsername(generateConstellationName())
	player.UpdatePosition(randomPosition())

	log.Printf("creating new bot: %v", player.PlayerID)

	return &Bot{
		player: player,
	}
}

func (b *Bot) Start(w *World) {
	for {
		if b.player.disconnect {
			return
		}
		// check colisions
		b.checkColision(w)

		// pathfind
		if b.target == nil || b.steps > 25 {
			b.performPathfinding(w)
		} else {
			b.moveTowards(w, b.target)
			b.steps++
		}

		time.Sleep(60 * time.Millisecond)
	}
}

func (b *Bot) surface() uint32 {
	return uint32(math.Pi * float64(b.player.Radius) * float64(b.player.Radius) / 1000)
}

func (b *Bot) checkColision(w *World) {
	// check colisions with food
	surface := b.surface()
	w.foodMutex.RLock()
	for _, food := range w.food {
		if distance(b.player.Position, &food.position) < surface {
			// dist := distance(b.player.Position, &food.position)
			newRadius := uint32(math.Sqrt(float64(b.player.Radius) * float64(b.player.Radius) + 30*30) * 1.0002);
			w.foodMutex.RUnlock()
			w.operationPlayerEatFood(b.player, &proto.EatFoodOperation{
				FoodPosition: food.position.toPacket(),
				NewRadius:    &newRadius,
			})
			b.target = nil
			return
		}
	}
	w.foodMutex.RUnlock()

	w.playersMutex.RLock()
	for _, player := range w.players {
		if player.PlayerID == b.player.PlayerID || player.Radius <= b.player.Radius + 5 {
			continue
		}
		if distance(b.player.Position, player.Position) < surface {
			newRadius := b.player.Radius + player.Radius
			w.playersMutex.RUnlock()
			w.operationEatPlayer(b.player, &proto.EatPlayerOperation{
				PlayerEaten: player.PlayerID[:],
				NewRadius:   &newRadius,
			})
			b.target = nil
			return
		}
	}
	w.playersMutex.RUnlock()
}

func (b *Bot) performPathfinding(w *World) {
	var foodTargets []*Vector2D
	var playerTargets []*Vector2D

	w.foodMutex.RLock()
	for _, food := range w.food {
		dist := distance(b.player.Position, &food.position)
		if dist < MAX_RANGE {
			foodTargets = append(foodTargets, &food.position)
		}
	}
	w.foodMutex.RUnlock()

	w.playersMutex.RLock()
	for _, player := range w.players {
		if player.PlayerID == b.player.PlayerID || player.Radius > b.player.Radius {
			continue
		}
		dist := distance(b.player.Position, player.Position)
		if dist < MAX_RANGE {
			playerTargets = append(playerTargets, player.Position)
		}
	}
	w.playersMutex.RUnlock()

	if len(playerTargets) == 0 && len(foodTargets) == 0 {
		return
	}

	bestTarget := foodTargets[0]
	for _, target := range foodTargets {
		if distance(b.player.Position, target) < distance(b.player.Position, bestTarget) {
			bestTarget = target
		}
	}

	for _, target := range playerTargets {
		dist := int32(distance(b.player.Position, target))
		if (dist - PLAYER_PREFERANCE) < int32(distance(b.player.Position, bestTarget)) {
			bestTarget = target
		}
	}

	b.target = bestTarget
	b.steps = 0
}

func (b *Bot) moveTowards(w *World, target *Vector2D) {
	var newX uint32
	var newY uint32

	deltaX := int32(b.player.Position.X) - int32(target.X)

	if math.Abs(float64(deltaX)) > SPEED {
		if deltaX < 0 {
			newX = b.player.Position.X + SPEED
		} else {
			newX = b.player.Position.X - SPEED
		}
	} else {
		newX = target.X
	}

	deltaY := int32(b.player.Position.Y) - int32(target.Y)

	if math.Abs(float64(deltaY)) > SPEED {
		if deltaY < 0 {
			newY = b.player.Position.Y + SPEED
		} else {
			newY = b.player.Position.Y - SPEED
		}
	} else {
		newY = target.Y
	}

	newPosition := Vector2D{
		X: newX,
		Y: newY,
	}

	if newPosition.X == target.X && newPosition.Y == target.Y {
		b.target = nil
	}

	// log.Printf("moving bot %v from %v to %v, target=%v, deltaX=%v deltaY=%v", b.player.PlayerID.String(), b.player.Position, newPosition, target, deltaX, deltaY)
	w.operationPlayerMove(b.player, &proto.MoveOperation{
		Position: newPosition.toPacket(),
	})

}

func distance(v1 *Vector2D, v2 *Vector2D) uint32 {
	dx := float64(v1.X) - float64(v2.X)
	dy := float64(v1.Y) - float64(v2.Y)
	dist := math.Abs(dx) + math.Abs(dy)
	return uint32(dist)
}

const (
	MAX_RANGE = 1100
	SPEED     = 10
	PLAYER_PREFERANCE = 500
	// FOOD_SURFACE = math.Pi*30*30
)
