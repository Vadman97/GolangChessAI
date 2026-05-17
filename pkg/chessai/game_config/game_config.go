package game_config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"
)

type GameConfiguration struct {
	Algorithm                 string
	MovesToPlay               int32
	SecondsToPlay             time.Duration
	AIMaxSearchDepth          int
	AIMaxThinkTimeMs          time.Duration
	AIScaleThinkTimeWithHuman bool
}

const FilePath = "game_conf.json"

var cfg *GameConfiguration

func findConfigFile(name string) string {
	dir, _ := os.Getwd()
	for {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func Get() *GameConfiguration {
	if cfg == nil {
		candidates := []string{
			FilePath,
			findConfigFile(FilePath),
			filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "Vadman97", "GolangChessAI", FilePath),
		}
		var file *os.File
		for _, p := range candidates {
			if p == "" {
				continue
			}
			f, err := os.Open(p)
			if err == nil {
				file = f
				break
			}
		}
		if file == nil {
			log.Panic("configuration parsing failed: could not open ", FilePath)
		}
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
