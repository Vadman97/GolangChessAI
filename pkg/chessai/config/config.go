package config

import (
	"encoding/json"
	"log"
	"os"
)

type Configuration struct {
	MemoryLimit                uint64
	TestRandSeed               int64
	UseOpenings                bool
	CacheGetAllMoves           bool
	CacheGetAllAttackableMoves bool
	TranspositionTableEnabled  bool
}

const FilePath = "conf.json"

var cfg *Configuration

func Get() *Configuration {
	if cfg == nil {
		file, _ := os.Open(FilePath)
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
