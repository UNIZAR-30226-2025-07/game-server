package galaxy

import (
	"bytes"
	"log"
	"math/rand/v2"
	"net/http"
	"sync"

	pb "galaxy.io/server/proto"
	"github.com/google/uuid"
)

const (
	WORLD_WIDTH  = 10_000
	WORLD_HEIGHT = 10_000
)

// PlayerID is a UUID v4 identifying a unique player.
// This identifier will be shared with the database.
type PlayerID uuid.UUID

type Vector2D struct {
	X uint32
	Y uint32
}

func (v *Vector2D) toPacket() *pb.Vector2D {
	return &pb.Vector2D{
		X: &v.X,
		Y: &v.Y,
	}
}

func VectorFromPacket(packet *pb.Vector2D) *Vector2D {
	return &Vector2D{
		X: *packet.X,
		Y: *packet.Y,
	}
}

func randomPosition() *Vector2D {
	return &Vector2D{
		X: rand.Uint32N(WORLD_WIDTH),
		Y: rand.Uint32N(WORLD_HEIGHT),
	}
}

// World holds all elements inside a current game, this includes players, bots and food.
// World is locked behind a mutex in order to archieve safe concurrency.
// Each server should only contain one world at the moment.
type World struct {
	sync.RWMutex
	food              []Food
	foodMutex         sync.RWMutex
	players           map[uuid.UUID]*Player
	playersMutex      sync.RWMutex
	connectionFactory ConnectionFactory
}

func NewWorld(factory ConnectionFactory) *World {
	return &World{
		players:           make(map[uuid.UUID]*Player),
		connectionFactory: factory,
	}
}

func (w *World) HandleNewConnection(writer http.ResponseWriter, r *http.Request) {
	playerID := uuid.New()

	operationHandler := func(operation *pb.Operation) {
		w.handlePlayerOperation(playerID, operation)
	}

	conn, err := w.connectionFactory.NewConnection(writer, r, operationHandler)
	if err != nil {
		log.Printf("Error creating connection: %v", err)
		return
	}

	player := NewPlayer(playerID, conn)

	w.registerPlayer(player)
	w.sendState(player)
	w.broadcastNewPlayer(player)
}

func (w *World) broadcastEvent(event *pb.Event) {
	w.playersMutex.RLock()
	defer w.playersMutex.RUnlock()

	for _, player := range w.players {
		err := player.SendEvent(event)

		if err != nil {
			log.Printf("error sending event %v to player %v", event, player.PlayerID.String())
		}
	}
}

func (w *World) registerPlayer(player *Player) {
	w.playersMutex.Lock()
	w.players[player.PlayerID] = player
	w.playersMutex.Unlock()
}

func (w *World) removePlayer(player *Player) {
	w.playersMutex.Lock()

	if _, exists := w.players[player.PlayerID]; !exists {
		w.playersMutex.Unlock()
		return
	}

	player.Disconnect()
	delete(w.players, player.PlayerID)
	w.playersMutex.Unlock()

	// broadcast player left event
	event := &pb.Event{
		EventType: pb.EventType_EvDestroyPlayer.Enum(),
		EventData: &pb.Event_DestroyPlayerEvent{
			DestroyPlayerEvent: &pb.DestroyPlayerEvent{
				PlayerID: player.PlayerID[:],
			},
		},
	}

	w.broadcastEvent(event)
}

func (w *World) broadcastNewPlayer(player *Player) {
	event := &pb.Event{
		EventType: pb.EventType_EvNewPlayer.Enum(),
		EventData: &pb.Event_NewPlayerEvent{
			NewPlayerEvent: &pb.NewPlayerEvent{
				PlayerID: player.PlayerID[:],
				Position: player.Position.toPacket(),
				Radius:   &player.Radius,
				Color:    &player.Skin,
			},
		},
	}

	w.broadcastEvent(event)
}

func (w *World) sendState(receiver *Player) {
	w.playersMutex.RLock()
	defer w.playersMutex.Unlock()

	for _, player := range w.players {
		if player.PlayerID == receiver.PlayerID {
			continue
		}

		event := &pb.Event{
			EventType: pb.EventType_EvNewPlayer.Enum(),
			EventData: &pb.Event_NewPlayerEvent{
				NewPlayerEvent: &pb.NewPlayerEvent{
					PlayerID: player.PlayerID[:],
					Position: player.Position.toPacket(),
					Radius:   &player.Radius,
					Color:    &player.Skin,
				},
			},
		}

		receiver.SendEvent(event)
	}

	for _, food := range w.food {
		event := &pb.Event{
			EventType: pb.EventType_EvNewFood.Enum(),
			EventData: &pb.Event_NewFoodEvent{
				NewFoodEvent: &pb.NewFoodEvent{
					Position: food.position.toPacket(),
					Color:    &food.color,
				},
			},
		}

		receiver.SendEvent(event)
	}
}

/// OPERATIONS

func (w *World) handlePlayerOperation(playerID uuid.UUID, operation *pb.Operation) {
	w.playersMutex.RLock()
	player, exists := w.players[playerID]
	w.playersMutex.RUnlock()

	if !exists {
		return
	}

	// Check the player is the author of the event
	author, err := playerID.MarshalBinary()
	if err != nil {
		log.Printf("invalid uuid: %v", err)
		return
	}

	if !bytes.Equal(author, operation.PlayerID) {
		log.Printf("the player %v tried to move a different player", player.PlayerID)
		return
	}

	switch operation.OperationType {
	case pb.OperationType_OpMove.Enum():
		w.operationPlayerMove(player, operation.GetMoveOperation())
	case pb.OperationType_OpEatFood.Enum():
		w.operationPlayerEatFood(player, operation.GetEatFoodOperation())
	case pb.OperationType_OpEatPlayer.Enum():
		w.operationEatPlayer(player, operation.GetEatPlayerOperation())
	case pb.OperationType_OpLeave.Enum():
		w.removePlayer(player)
	default:
		log.Printf("unimplemented event: %v", operation.OperationType.Enum().String())
		return
	}
}

func (w *World) operationPlayerMove(player *Player, moveOperation *pb.MoveOperation) {
	// TODO: check for cheaters
	player.UpdatePosition(VectorFromPacket(moveOperation.Position))

	// broadcast the movement to all the players
	playerIDBytes, _ := player.PlayerID.MarshalBinary()
	moveEvent := &pb.Event{
		EventType: pb.EventType_EvPlayerMove.Enum(),
		EventData: &pb.Event_PlayerMoveEvent{
			PlayerMoveEvent: &pb.PlayerMoveEvent{
				PlayerID: playerIDBytes,
				Position: player.Position.toPacket(),
			},
		},
	}

	w.broadcastEvent(moveEvent)
}

func (w *World) operationPlayerEatFood(player *Player, operation *pb.EatFoodOperation) {
	player.UpdateRadius(*operation.NewRadius)

	playerIDBytes, _ := player.PlayerID.MarshalBinary()
	eventGrow := &pb.Event{
		EventType: pb.EventType_EvPlayerGrow.Enum(),
		EventData: &pb.Event_PlayerGrowEvent{
			PlayerGrowEvent: &pb.PlayerGrowEvent{
				PlayerID: playerIDBytes,
				Radius:   operation.NewRadius,
			},
		},
	}

	eventFoodDestroy := &pb.Event{
		EventType: pb.EventType_EvDestroyFood.Enum(),
		EventData: &pb.Event_DestroyFoodEvent{
			DestroyFoodEvent: &pb.DestroyFoodEvent{
				Position: operation.FoodPosition,
			},
		},
	}

	w.broadcastEvent(eventGrow)
	w.broadcastEvent(eventFoodDestroy)
}

func (w *World) operationEatPlayer(player *Player, operation *pb.EatPlayerOperation) {
	player.UpdateRadius(*operation.NewRadius)

	playerIDBytes, _ := player.PlayerID.MarshalBinary()
	eventGrow := &pb.Event{
		EventType: pb.EventType_EvPlayerGrow.Enum(),
		EventData: &pb.Event_PlayerGrowEvent{
			PlayerGrowEvent: &pb.PlayerGrowEvent{
				PlayerID: playerIDBytes,
				Radius:   operation.NewRadius,
			},
		},
	}

	eventDestroyPlayer := &pb.Event{
		EventType: pb.EventType_EvDestroyPlayer.Enum(),
		EventData: &pb.Event_DestroyPlayerEvent{
			DestroyPlayerEvent: &pb.DestroyPlayerEvent{
				PlayerID: operation.PlayerEaten,
			},
		},
	}

	w.broadcastEvent(eventGrow)
	w.broadcastEvent(eventDestroyPlayer)

	playerEatenID, _ := uuid.FromBytes(operation.PlayerEaten)
	w.playersMutex.Lock()
	playerEaten, exists := w.players[playerEatenID]
	w.playersMutex.Unlock()
	if !exists {
		return
	}

	w.removePlayer(playerEaten)
	w.broadcastEvent(eventDestroyPlayer)
	w.broadcastEvent(eventGrow)
}
