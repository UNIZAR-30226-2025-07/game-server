package galaxy

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
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
	Quantity uint64 `json:"quantity"`
}

func newDatabase() *Database {
	proxyURL, _ := url.Parse("http://localhost:9999")
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	return &Database{
		httpClient: &http.Client{
			Transport: transport,
		},
	}
}

func (d *Database) PostAchievements(player *Player) {
	player.Stats.Lock()
	defer player.Stats.Unlock()

	scoreData := postData{
		UserID:   player.PlayerID.String(),
		Kind:     "maxScore",
		Quantity: uint64(player.Stats.Score),
	}

	killData := postData{
		UserID:   player.PlayerID.String(),
		Kind:     "playersEliminated",
		Quantity: uint64(player.Stats.KilledPlayers),
	}

	timePlayed := player.Stats.TimeEnd.Sub(player.Stats.TimeStart)
	timeData := postData{
		UserID:   player.PlayerID.String(),
		Kind:     "timePlayed",
		Quantity: uint64(timePlayed.Seconds()),
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
