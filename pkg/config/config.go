package config

import (
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
)

type Config struct {
	APIBindPort          int
	APIExternalURL       string
	LevelDBDatabasesPath string
	RootToken            string
}

func FromEnv() Config {
	godotenv.Load()
	bindPort, err := strconv.Atoi(os.Getenv("api_bind_port"))
	if err != nil {
		log.Fatal(err)
	}
	return Config{
		APIBindPort:          bindPort,
		APIExternalURL:       os.Getenv("api_url"),
		LevelDBDatabasesPath: os.Getenv("leveldb_databases_path"),
		RootToken:            os.Getenv("root_token"),
	}
}
