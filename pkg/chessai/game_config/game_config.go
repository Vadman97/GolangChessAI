package game_config

import (
	"encoding/json"
	"log"
	"os"
	"path"
	"time"
)

type GameConfiguration struct {
	Algorithm   string
	MovesToPlay int32
	TimeToPlay  time.Duration
}

const FilePath = "game_config.json"

var cfg *GameConfiguration

func Get() *GameConfiguration {
	if cfg == nil {
		dir := path.Join(os.Getenv("GOPATH"), "src", "github.com", "Vadman97", "ChessAI3", FilePath)
		file, _ := os.Open(dir)
		defer func() { _ = file.Close() }()
		decoder := json.NewDecoder(file)
		configuration := GameConfiguration{}
		err := decoder.Decode(&configuration)
		if err != nil {
			log.Panic("configuration parsing failed ", err)
		}
		cfg = &configuration
	}
	return cfg
}
