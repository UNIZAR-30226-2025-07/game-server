package galaxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	URL = "http://galaxy.t2dc.es:3000"
)

type Database struct {
	httpClient *http.Client
}

type postData struct {
	UserID   string `json:"user_id"`
	Kind     string `json:"achievement_type"`
	Quantity uint32 `json:"quantity"`
}

func newDatabase() *Database {
	return &Database{
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

type startPrivateGameData struct {
	GameID uint32 `json:"gameId"`
}

func (d *Database) StartPrivateGame(gameID uint32) {
	data := startPrivateGameData{
		GameID: gameID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error while marshaling startPrivateGameData: %v", data)
		return
	}

	resp, err := d.httpClient.Post(URL+"/private/startPrivateGame", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error while sending startPrivateGame: %v, err: %v", data, err)
	} else {
		if resp.StatusCode != 200 {
			log.Printf("Bad response code while sending startPrivateGame: %v, code: %v", data, resp.StatusCode)
		}
	}
}

type pausePrivateGameData struct {
	GameID uint32 `json:"gameId"`
}

func (d *Database) PausePrivateGame(gameID uint32) {
	data := pausePrivateGameData{
		GameID: gameID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error while marshaling pausePrivateGameData: %v", data)
		return
	}

	resp, err := d.httpClient.Post(URL+"/private/pausePrivateGame", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error while sending pausePrivateGame: %v, err: %v", data, err)
	} else {
		if resp.StatusCode != 200 {
			log.Printf("Bad response code while sending pausePrivateGame: %v, code: %v", data, resp.StatusCode)
		}
	}
}

type getValuesData struct {
	GameID uint32 `json:"gameId"`
}

type PlayerData struct {
	PlayerID string `json:"id_user"`
	X        uint32 `json:"x_position"`
	Y        uint32 `json:"y_position"`
	Score    uint32 `json:"score"`
}

func (d *Database) GetValues(gameID uint32) []PlayerData {
	resp, err := d.httpClient.Get(URL + "/private/getValues/" + strconv.FormatUint(uint64(gameID), 10))
	if err != nil {
		log.Printf("Error while sending getValues: %v, err: %v", gameID, err)
		return nil
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Bad response code while sending getValues: %v, code: %v", gameID, resp.StatusCode)
		return nil
	}

	responseBody, _ := io.ReadAll(resp.Body)

	var gameData []PlayerData
	err = json.Unmarshal(responseBody, &gameData)
	if err != nil {
		log.Printf("ERROR: unmarshing gameData, check proxy. err: %v", err)
		return nil
	}

	log.Printf("len gameData: %v", len(gameData))
	for _, player := range gameData {
		log.Printf("player gotten from gameData: id=%v x=%v y=%v score=%v", player.PlayerID, player.X, player.Y, player.Score)
	}

	return gameData
}

func (d *Database) UpdateValues(w *World) {
	if w.gameID == nil {
		log.Printf("ERROR: tried uploading match to database in a public match. FIXME FIXME FIXME")
		return
	}
	// players
	log.Printf("uploading match to database, len = %v", len(w.players))
	var gameData []PlayerData
	for _, player := range w.players {
		gameData = append(gameData, PlayerData{
			PlayerID: player.PlayerID.String(),
			X:        player.Position.X,
			Y:        player.Position.Y,
			Score:    uint32(player.Radius / 10),
		})
	}

	jsonData, err := json.Marshal(gameData)
	if err != nil {
		log.Printf("Error while marshaling gameData: %v", gameData)
		return
	}

	resp, err := d.httpClient.Post(URL+"/private/uploadValues/"+strconv.FormatUint(uint64(*w.gameID), 10), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error while sending updateValues: %v, err: %v", gameData, err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Bad response code while sending updateValues: %v, code: %v", gameData, resp.StatusCode)
		return
	}
}

func (d *Database) PostAchievements(player *Player) {
	player.Stats.Lock()
	defer player.Stats.Unlock()

	scoreData := postData{
		UserID:   player.PlayerID.String(),
		Kind:     "maxScore",
		Quantity: player.Stats.Score,
	}

	killData := postData{
		UserID:   player.PlayerID.String(),
		Kind:     "playersEliminated",
		Quantity: player.Stats.KilledPlayers,
	}

	timePlayed := player.Stats.TimeEnd.Sub(player.Stats.TimeStart)
	timeData := postData{
		UserID:   player.PlayerID.String(),
		Kind:     "timePlayed",
		Quantity: uint32(timePlayed.Seconds()),
	}

	scoreJsonData, err := json.Marshal(scoreData)
	if err != nil {
		log.Printf("Error while marshaling scoreData: %v", scoreData)
	} else {
		resp, err := d.httpClient.Post(URL+"/achievements/update-achievement", "application/json", bytes.NewBuffer(scoreJsonData))
		if err != nil {
			log.Printf("Error while sending scoreData: %v, err: %v", scoreData, err)
		} else {
			if resp.StatusCode != 200 {
				log.Printf("Bad response code while sending scoreData: %v, code: %v", scoreData, resp.StatusCode)
			}
		}
	}

	killJsonData, err := json.Marshal(killData)
	if err != nil {
		log.Printf("Error while marshaling killData: %v", killData)
	} else {
		resp, err := d.httpClient.Post(URL+"/achievements/update-achievement", "application/json", bytes.NewBuffer(killJsonData))
		if err != nil {
			log.Printf("Error while sending killData: %v", killData)
		} else {
			if resp.StatusCode != 200 {
				log.Printf("Bad response code while sending killData: %v, code: %v", killData, resp.StatusCode)
			}
		}
	}

	timeJsonData, err := json.Marshal(timeData)
	if err != nil {
		log.Printf("Error while marshaling timeData: %v", timeData)
	} else {
		resp, err := d.httpClient.Post(URL+"/achievements/update-achievement", "application/json", bytes.NewBuffer(timeJsonData))
		if err != nil {
			log.Printf("Error while sending timeData: %v", timeData)
		} else {
			if resp.StatusCode != 200 {
				log.Printf("Bad response code while sending timeData: %v, code: %v", timeData, resp.StatusCode)
			}
		}
	}
}
