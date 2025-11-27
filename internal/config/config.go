package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost          string
	DBUser          string
	DBPassword      string
	DBName          string
	DBPort          string
	AppPort         string
	XenditSecretKey string
	AppEnv          string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		DBHost:          os.Getenv("DB_HOST"),
		DBUser:          os.Getenv("DB_USER"),
		DBPassword:      os.Getenv("DB_PASSWORD"),
		DBName:          os.Getenv("DB_NAME"),
		DBPort:          os.Getenv("DB_PORT"),
		AppPort:         os.Getenv("APP_PORT"),
		XenditSecretKey: os.Getenv("XENDIT_APIKEY"),
		AppEnv:          os.Getenv("APP_ENV"),
	}

	if cfg.DBHost == "" {
		log.Fatal("Environment variables not loaded properly")
	}

	return cfg
}
