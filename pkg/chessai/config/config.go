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
	CacheMaxPlayerElements     uint64
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
	RandomMoveOrder            bool
	IterativeIncrement         int
}

const FilePath = "conf.json"

var cfg *Configuration

func Get() *Configuration {
	if cfg == nil {
		candidates := []string{
			FilePath,
			path.Join(os.Getenv("GOPATH"), "src", "github.com", "Vadman97", "GolangChessAI", FilePath),
		}
		var file *os.File
		for _, p := range candidates {
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
		configuration := Configuration{}
		err := decoder.Decode(&configuration)
		if err != nil {
			log.Panic("configuration parsing failed ", err)
		}
		cfg = &configuration
	}
	return cfg
}
