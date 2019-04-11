package config

import (
	"encoding/json"
	"log"
	"os"
	"path"
)

type Configuration struct {
	MemoryLimit                uint64
	TestRandSeed               int64
	UseOpenings                bool
	CacheGetAllMoves           bool
	CacheGetAllAttackableMoves bool
	TranspositionTableEnabled  bool
	LogDebug                   bool
	DebugLogFileName           string
	LogPerformance             bool
	PerformanceLogFileName     string
	LogPerformanceToExcel      bool
	ExcelPerformanceFileName   string
	PrintPlayerInfo            bool
	NumberOfCompetitionGames   int
	StartingElo                int
}

const FilePath = "conf.json"

var cfg *Configuration

func Get() *Configuration {
	if cfg == nil {
		dir := path.Join(os.Getenv("GOPATH"), "src", "github.com", "Vadman97", "ChessAI3", FilePath)
		file, _ := os.Open(dir)
		defer func() { _ = file.Close() }()
		decoder := json.NewDecoder(file)
		configuration := Configuration{}
		err := decoder.Decode(&configuration)
		if err != nil {
			log.Panic("configuration parsing failed ", err)
		}
		cfg = &configuration
	}
	return cfg
}
