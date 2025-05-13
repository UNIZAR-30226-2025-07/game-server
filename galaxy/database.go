package galaxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
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
		httpClient: &http.Client{},
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
	X uint32 `json:"x_position"`
	Y uint32 `json:"y_position"`
	Score uint32 `json:"score"`
}

func (d *Database) GetValues(gameID uint32) []PlayerData {
	data := getValuesData{
		GameID: gameID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error while marshaling getValuesData: %v", data)
		return nil
	}

	resp, err :=d.httpClient.Post(URL+"/private/getValues", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error while sending getValues: %v, err: %v", data, err)
		return nil
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Bad response code while sending getValues: %v, code: %v", data, resp.StatusCode)
		return nil
	}

	responseBody, _ := io.ReadAll(resp.Body)

	var gameData []PlayerData
	err = json.Unmarshal(responseBody, &gameData)
	if err != nil {
		log.Printf("ERROR: unmarshing gameData, check proxy. err: %v", err)
		return nil
	}

	return gameData
}

func (d *Database) UpdateValues() {
	log.Printf("updating values (NOT IMPLEMENTED YET)")
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
