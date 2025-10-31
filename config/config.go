package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Env            string
	PostgresURL    string
	RedisURL       string
	Debug          bool
	LogLevel       string
	FrontendOrigin string
	MigrationsPath string
}

var Global *Config

func InitGlobalConfig() {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Warning: cannot load %s, using system env vars")
	}
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "prod"
	}
	println(env)
	var envFile string
	switch env {
	case "prod":
		envFile = ".env.prod"
	default:
		envFile = ".env.dev"
	}

	if err := godotenv.Load(envFile); err != nil {
		log.Printf("Warning: cannot load %s, using system env vars", envFile)
	}
	Global = &Config{
		Env:            env,
		PostgresURL:    os.Getenv("POSTGRES_URL"),
		RedisURL:       os.Getenv("REDIS_URL"),
		Debug:          env != "prod",
		LogLevel:       os.Getenv("LOG_LEVEL"),
		FrontendOrigin: os.Getenv("FRONTEND_ORIGIN"),
		MigrationsPath: os.Getenv("MIGRATIONS_PATH"),
	}
}

func GetConfig() *Config {
	return Global
}
