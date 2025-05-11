package galaxy

import (
	"log"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"galaxy.io/server/proto"
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
	var x uint32
	var y uint32

	if packet.X == nil {
		x = 0
	} else {
		x = *packet.X
	}

	if packet.Y == nil {
		y = 0
	} else {
		y = *packet.Y
	}

	return &Vector2D{
		X: x,
		Y: y,
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
	playersConnection map[uuid.UUID]*Player
	playersMutex      sync.RWMutex
	connectionFactory ConnectionFactory
	database          *Database
}

func NewWorld(factory ConnectionFactory) *World {
	return &World{
		players:           make(map[uuid.UUID]*Player),
		playersConnection: make(map[uuid.UUID]*Player),
		food:              createRandomFood(),
		connectionFactory: factory,
		database:          newDatabase(),
	}
}

func (w *World) sendEvent(player *Player, event *pb.Event) {
	err := player.SendEvent(event)
	if err != nil {
		log.Printf("deleting player %v", player.PlayerID.String())
		w.removePlayer(player)
	}
}

func (w *World) HandleNewConnection(writer http.ResponseWriter, r *http.Request) {
	connectionID := uuid.New()
	log.Printf("handling new connection, id = %v", connectionID)

	operationHandler := func(operation *pb.Operation) {
		w.handlePlayerOperation(connectionID, operation)
	}

	conn, err := w.connectionFactory.NewConnection(writer, r, operationHandler)
	if err != nil {
		log.Printf("Error creating connection: %v", err)
		return
	}

	player := NewPlayer(connectionID, conn)
	//
	w.registerPlayer(player)
}

func (w *World) broadcastEvent(event *pb.Event) {
	w.playersMutex.RLock()
	defer w.playersMutex.RUnlock()

	for _, player := range w.players {
		if *event.EventType != proto.EventType_EvPlayerMove {
			log.Printf("sending event: %v to %v", event.EventType.String(), player.PlayerID.String())
		}

		w.sendEvent(player, event)
	}
}

func (w *World) registerPlayer(player *Player) {
	w.playersMutex.Lock()
	log.Printf("registering player with connection: %v", player.ConnectionID.String())
	w.playersConnection[player.ConnectionID] = player
	w.playersMutex.Unlock()
}

func (w *World) removePlayer(player *Player) {
	log.Printf("removing player: %v", player.PlayerID.String())
	w.playersMutex.Lock()
	defer w.playersMutex.Unlock()

	if _, exists := w.players[player.PlayerID]; !exists {
		return
	}

	player.Disconnect()
	delete(w.players, player.PlayerID)

	// broadcast player left event
	event := &pb.Event{
		EventType: pb.EventType_EvDestroyPlayer.Enum(),
		EventData: &pb.Event_DestroyPlayerEvent{
			DestroyPlayerEvent: &pb.DestroyPlayerEvent{
				PlayerID: player.PlayerID[:],
			},
		},
	}

	go w.broadcastEvent(event)

	// TODO: send stats to database
	player.Stats.Lock();
	player.Stats.TimeEnd = time.Now();
	player.Stats.Unlock();
	w.database.PostAchievements(player)
}

func (w *World) broadcastNewPlayer(player *Player) {
	event := &pb.Event{
		EventType: pb.EventType_EvNewPlayer.Enum(),
		EventData: &pb.Event_NewPlayerEvent{
			NewPlayerEvent: &pb.NewPlayerEvent{
				PlayerID: player.PlayerID[:],
				Position: player.Position.toPacket(),
				Radius:   &player.Radius,
				Color:    &player.Color,
				Skin:     player.Skin,
			},
		},
	}

	go w.broadcastEvent(event)
}

func (w *World) sendJoin(player *Player) {
	w.playersMutex.RLock()
	defer w.playersMutex.RUnlock()

	event := &pb.Event{
		EventType: pb.EventType_EvJoin.Enum(),
		EventData: &pb.Event_JoinEvent{
			JoinEvent: &pb.JoinEvent{
				PlayerID: player.PlayerID[:],
				Position: player.Position.toPacket(),
				Radius:   &player.Radius,
				Color:    &player.Color,
				Skin:     player.Skin,
			},
		},
	}

	log.Printf("sending join")
	w.sendEvent(player, event)
}

func (w *World) sendState(receiver *Player) {
	w.playersMutex.RLock()
	defer w.playersMutex.RUnlock()

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
					Color:    &player.Color,
					Skin:     player.Skin,
				},
			},
		}

		log.Printf("sending state %v to player %v", player.PlayerID, receiver.PlayerID)

		w.sendEvent(receiver, event)
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

		w.sendEvent(receiver, event)
	}
}

/// OPERATIONS

func (w *World) handlePlayerOperation(connectionID uuid.UUID, operation *pb.Operation) {
	// log.Printf("handling new operation, connection = %v, op = %v", connectionID, operation)
	w.playersMutex.RLock()
	player, exists := w.playersConnection[connectionID]
	w.playersMutex.RUnlock()

	if !exists {
		log.Printf("...does not exists")
		return
	}

	switch *operation.OperationType {
	case pb.OperationType_OpJoin:
		w.operationJoin(player, operation.GetJoinOperation())
	case pb.OperationType_OpMove:
		w.operationPlayerMove(player, operation.GetMoveOperation())
	case pb.OperationType_OpEatFood:
		w.operationPlayerEatFood(player, operation.GetEatFoodOperation())
	case pb.OperationType_OpEatPlayer:
		w.operationEatPlayer(player, operation.GetEatPlayerOperation())
	case pb.OperationType_OpLeave:
		w.removePlayer(player)
	default:
		log.Printf("unimplemented event: %v", operation.OperationType.Enum().String())
		return
	}
}

func (w *World) operationJoin(player *Player, joinOperation *pb.JoinOperation) {
	playerID, err := uuid.FromBytes(joinOperation.PlayerID)
	if err != nil {
		log.Printf("warn: unable to parse playerID: %v", err)
		return
	}
	log.Printf("joining with id: %v", playerID.String())
	player.UpdatePlayerID(playerID)
	player.UpdateUsername(*joinOperation.Username)
	player.UpdateColor(*joinOperation.Color)
	if joinOperation.Skin != nil {
		player.UpdateSkin(*joinOperation.Skin)
	}

	w.Lock()
	w.players[player.PlayerID] = player
	w.Unlock()

	w.sendJoin(player)
	go w.sendState(player)
	go w.broadcastNewPlayer(player)

	player.Stats.Lock()
	player.Stats.TimeStart = time.Now()
	player.Stats.Unlock()
}

func (w *World) operationPlayerMove(player *Player, moveOperation *pb.MoveOperation) {
	if moveOperation == nil {
		log.Printf("nil operation in playerMove, player = %v", player.PlayerID.String())
		return
	}
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

	go w.broadcastEvent(moveEvent)
}

func (w *World) operationPlayerEatFood(player *Player, operation *pb.EatFoodOperation) {
	log.Printf("operationPlayerEatFood, player = %v, operation = %v", player.PlayerID.String(), operation.String())
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

	go w.broadcastEvent(eventGrow)
	go w.broadcastEvent(eventFoodDestroy)
}

func (w *World) operationEatPlayer(player *Player, operation *pb.EatPlayerOperation) {
	log.Printf("operationEatPlayer, player = %v, operation = %v", player.PlayerID.String(), operation.String())
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

	go w.broadcastEvent(eventGrow)
	go w.broadcastEvent(eventDestroyPlayer)

	playerEatenID, _ := uuid.FromBytes(operation.PlayerEaten)
	w.playersMutex.Lock()
	playerEaten, exists := w.players[playerEatenID]
	w.playersMutex.Unlock()
	if !exists {
		return
	}

	go w.removePlayer(playerEaten)
	go w.broadcastEvent(eventDestroyPlayer)
	go w.broadcastEvent(eventGrow)

	player.Stats.Lock()
	player.Stats.KilledPlayers++
	player.Stats.Unlock()
}
