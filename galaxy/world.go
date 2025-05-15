package galaxy

import (
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

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

func isPrivateServer() bool {
	value, exists := os.LookupEnv("PRIVATE_SERVER")

	if !exists {
		log.Printf("Starting as a public server.")
		return false // Assume false if the environment variable is not set
	}

	// Convert the value to lowercase for case-insensitive comparison
	lowerValue := strings.ToLower(value)

	// Check for common true values
	switch lowerValue {
	case "true", "1", "yes":
		log.Printf("Starting as a private server.")
		return true
	case "false", "0", "no":
		log.Printf("Starting as a public server.")
		return false
	default:
		log.Printf("We have no clue if you want a private server, going public.")
		return false
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
	privateServer     bool
	gameID            *uint32
	savedPlayers      []PlayerData
}

func NewWorld(factory ConnectionFactory) *World {
	return &World{
		players:           make(map[uuid.UUID]*Player),
		playersConnection: make(map[uuid.UUID]*Player),
		food:              createRandomFood(),
		connectionFactory: factory,
		database:          newDatabase(),
		privateServer:     isPrivateServer(),
	}
}

func (w *World) checkForBots() {
	for {
		time.Sleep(10 * time.Second)
		w.playersMutex.RLock()
		onlyBots := true
		for _, player := range w.players {
			if player.conn != nil {
				onlyBots = false
				break
			}
		}
		w.playersMutex.RUnlock()

		if onlyBots {
			return
		}

		if len(w.players) < 5 {
			log.Printf("less than 5 players in game, creating bot")
			bot := NewBot()
			w.players[bot.player.PlayerID] = bot.player
			w.broadcastNewPlayer(bot.player)
			go bot.Start(w)
		}
	}
}

func (w *World) sendEvent(player *Player, event *pb.Event) error {
	err := player.SendEvent(event)
	return err
	// if err != nil {
	// }
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
	w.registerPlayer(player)
}

func (w *World) broadcastEvent(event *pb.Event) {
	w.playersMutex.RLock()
	defer w.playersMutex.RUnlock()

	for _, player := range w.players {
		err := w.sendEvent(player, event)
		if err != nil {
			w.playersMutex.RUnlock()
			log.Printf("deleting player %v becau", player.PlayerID.String())
			w.removePlayer(player)
			w.playersMutex.RLock()
		}
	}
}

func (w *World) registerPlayer(player *Player) {
	w.playersMutex.Lock()
	w.playersConnection[player.ConnectionID] = player
	w.playersMutex.Unlock()
}

func (w *World) removePlayer(player *Player) {
	log.Printf("removing player: %v", player.PlayerID.String())
	w.playersMutex.Lock()

	if _, exists := w.players[player.PlayerID]; !exists {
		w.playersMutex.Unlock()
		return
	}

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
	time.Sleep(200 * time.Millisecond)
	player.Disconnect()
	player.Stats.TimeEnd = time.Now()
	w.database.PostAchievements(player)

	if w.privateServer && len(w.players) == 0 {
		log.Printf("restarting private server as no players are online")
		w.gameID = nil
	}
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
				Username: &player.Username,
			},
		},
	}

	w.broadcastEvent(event)
	w.playersMutex.Lock()
	time.Sleep(100 * time.Millisecond)
	w.playersMutex.Unlock()
}

func (w *World) sendJoin(player *Player) {
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

	err := w.sendEvent(player, event)
	if err != nil {
		log.Printf("deleting player %v becau", player.PlayerID.String())
		w.removePlayer(player)
	}
}

func (w *World) sendState(receiver *Player) {
	w.playersMutex.RLock()

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
					Username: &player.Username,
				},
			},
		}

		log.Printf("sending state %v to player %v", player.ConnectionID, receiver.ConnectionID)

		err := w.sendEvent(receiver, event)
		if err != nil {
			log.Printf("deleting player %v becau", player.PlayerID.String())
			w.playersMutex.RUnlock()
			w.removePlayer(receiver)
			w.playersMutex.RLock()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)

	var pbFoods []*pb.Food

	for _, food := range w.food {
		pbFoods = append(pbFoods, &pb.Food{
			Position: food.position.toPacket(),
			Color:    &food.color,
		})
	}
	event := &pb.Event{
		EventType: pb.EventType_EvNewFood.Enum(),
		EventData: &pb.Event_NewFoodEvent{
			NewFoodEvent: &pb.NewFoodEvent{
				Food: pbFoods,
			},
		},
	}

	w.sendEvent(receiver, event)
	time.Sleep(200 * time.Millisecond)
	w.playersMutex.RUnlock()
}

/// OPERATIONS

func (w *World) handlePlayerOperation(connectionID uuid.UUID, operation *pb.Operation) {
	if (*operation.OperationType != pb.OperationType_OpMove) {
		log.Printf("handling new operation, player = %v, op = %v", connectionID, operation)
	}
	w.playersMutex.RLock()
	player, exists := w.playersConnection[connectionID]
	w.playersMutex.RUnlock()

	if !exists {
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
	case pb.OperationType_OpPause:
		w.pauseServer()
	default:
		log.Printf("unimplemented event: %v", operation.OperationType.Enum().String())
		return
	}
}

func (w *World) pauseServer() {
	if w.gameID == nil {
		// pause is not implemented in public matches
		log.Printf("pausing in a public server")
		return
	}

	pauseEvent := &pb.Event{
		EventType: pb.EventType_EvPause.Enum(),
		EventData: &pb.Event_PauseEvent{},
	}

	log.Printf("broadcasting pause")
	w.broadcastEvent(pauseEvent)
	w.playersMutex.Lock()
	w.database.PausePrivateGame(*w.gameID)
	w.database.UpdateValues(w)

	for id, player := range w.players {
		player.Disconnect()
		delete(w.players, id)
	}
	w.playersMutex.Unlock()

	log.Printf("restarting private server")
	w.gameID = nil
}

func (w *World) operationJoin(player *Player, joinOperation *pb.JoinOperation) {
	log.Printf("player joined %v, data=%v", player, joinOperation)
	playerID, err := uuid.FromBytes(joinOperation.PlayerID)
	if err != nil {
		log.Printf("warn: unable to parse playerID: %v", err)
		return
	}
	player.UpdatePlayerID(playerID)
	player.UpdateUsername(*joinOperation.Username)
	player.UpdateColor(*joinOperation.Color)
	if joinOperation.Skin != nil {
		player.UpdateSkin(*joinOperation.Skin)
	}

	if w.privateServer {
		if joinOperation.GameID == nil {
			log.Printf("ERROR: a player tried joining a private server without gameID, kicking him.")
			return
		}

		if w.gameID == nil {
			w.gameID = joinOperation.GameID
			log.Printf("set up gameID: %v", *w.gameID)
			w.database.StartPrivateGame(*w.gameID)
			w.savedPlayers = w.database.GetValues(*w.gameID)
		} else {
			if *w.gameID != *joinOperation.GameID {
				log.Printf("ERROR: a player tried joining a private server with the wrong gameID, kicking him. gameID = %v", *joinOperation.GameID)
				return
			}
		}

		for _, savedPlayer := range w.savedPlayers {
			if savedPlayer.Score == 0 {
				continue
			}
			if savedPlayer.PlayerID == player.PlayerID.String() {
				player.UpdatePosition(&Vector2D{
					X: savedPlayer.X,
					Y: savedPlayer.Y,
				})
				player.UpdateRadius(savedPlayer.Score * 10)
				break
			}
		}
	}

	w.sendJoin(player)
	time.Sleep(200*time.Millisecond)
	w.sendState(player)

	w.playersMutex.Lock()
	if len(w.players) == 0 {
		// first player
		if !w.privateServer {
			// only in public matches
			go w.checkForBots()
		}
	}
	w.players[player.PlayerID] = player
	w.playersMutex.Unlock()

	w.broadcastNewPlayer(player)

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

	w.broadcastEvent(moveEvent)
}

func (w *World) operationPlayerEatFood(player *Player, operation *pb.EatFoodOperation) {
	player.UpdateRadius(*operation.NewRadius)

	foodPos := VectorFromPacket(operation.FoodPosition)
	w.foodMutex.Lock()
	for i, f := range w.food {
		if f.position == *foodPos {
			w.food = append(w.food[:i], w.food[i+1:]...)
			// add new food
			newFood := Food{
				position: *randomPosition(),
				color:    randomColor(),
			}
			w.food = append(w.food, newFood)

			newFoodEvent := &pb.Event{
				EventType: pb.EventType_EvNewFood.Enum(),
				EventData: &pb.Event_NewFoodEvent{
					NewFoodEvent: &pb.NewFoodEvent{
						Food: []*pb.Food{{
							Position: newFood.position.toPacket(),
							Color:    &newFood.color}}}},
			}

			w.broadcastEvent(newFoodEvent)
			break
		}
	}
	w.foodMutex.Unlock()

	playerIDBytes, _ := player.PlayerID.MarshalBinary()
	eventGrow := &pb.Event{
		EventType: pb.EventType_EvPlayerGrow.Enum(),
		EventData: &pb.Event_PlayerGrowEvent{
			PlayerGrowEvent: &pb.PlayerGrowEvent{
				PlayerID: playerIDBytes,
				Radius:   &player.Radius,
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
	time.Sleep(15*time.Millisecond)
	w.broadcastEvent(eventFoodDestroy)
}

func (w *World) operationEatPlayer(player *Player, operation *pb.EatPlayerOperation) {
	log.Printf("operationEatPlayer, player = %v, operation = %v", player.PlayerID.String(), operation.String())

	playerToEatID, _ := uuid.FromBytes(operation.PlayerEaten)
	w.playersMutex.RLock()
	playerToEat, exists := w.players[playerToEatID]
	w.playersMutex.RUnlock()
	if !exists {
		log.Printf("...trying to eat a dead player")
		return
	}

	if player.Radius <= playerToEat.Radius {
		log.Printf("player %v tried to eat player %v while being equal or smaller size", player.ConnectionID, playerToEat.ConnectionID)
		return
	}

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

	player.Stats.Lock()
	player.Stats.KilledPlayers++
	player.Stats.Unlock()
}
