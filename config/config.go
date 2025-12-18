package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Env                string
	PostgresURL        string
	RedisURL           string
	ElasticUser        string
	ElasticPassword    string
	ElasticPort        string
	ElasticIndexesPath string
	Debug              bool
	LogLevel           string
	FrontendOrigin     string
	MigrationsPath     string
	Origin             string
	AuthService        string
	FriendService      string
	ProfileService     string
	MainService        string
	PostgresURLMain    string
	PostgresURLFriend  string
	PostgresURLProfile string
	PostgresURLAuth    string
}

func NewConfig() *Config {

	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Warning: cannot load env)")
	}

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")
	mainDBUser := os.Getenv("MAIN_DB_USER")
	mainDBPassword := os.Getenv("MAIN_DB_PASSWORD")
	profileDBUser := os.Getenv("PROFILE_DB_USER")
	profileDBPassword := os.Getenv("PROFILE_DB_PASSWORD")
	authDBUser := os.Getenv("AUTH_DB_USER")
	authDBPassword := os.Getenv("AUTH_DB_PASSWORD")
	friendDBUser := os.Getenv("FRIEND_DB_USER")
	frinedDBPassword := os.Getenv("FRIEND_DB_PASSWORD")
	postgresURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode,
	)
	postgresURLMain := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		mainDBUser, mainDBPassword, dbHost, dbPort, dbName, dbSSLMode,
	)
	postgresURLAuth := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		authDBUser, authDBPassword, dbHost, dbPort, dbName, dbSSLMode,
	)
	postgresURLProfile := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		profileDBUser, profileDBPassword, dbHost, dbPort, dbName, dbSSLMode,
	)
	postgresURLFrined := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		friendDBUser, frinedDBPassword, dbHost, dbPort, dbName, dbSSLMode,
	)

	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := os.Getenv("REDIS_DB")
	redisURL := fmt.Sprintf("redis://:%s@%s:%s/%s", redisPassword, redisHost, redisPort, redisDB)
	origin := os.Getenv("ORIGIN")
	elasticUser := os.Getenv("ELASTIC_USER")
	elasticPassword := os.Getenv("ELASTIC_PASSWORD")
	elasticPort := os.Getenv("ELASTIC_PORT")
	elasticIndexesPath := os.Getenv("ELASTIC_INDEXES_PATH")

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	migrationsPath = fmt.Sprintf("file://%s", migrationsPath)

	config := &Config{
		Env:                os.Getenv("APP_ENV"),
		PostgresURL:        postgresURL,
		RedisURL:           redisURL,
		ElasticUser:        elasticUser,
		ElasticPassword:    elasticPassword,
		ElasticPort:        elasticPort,
		ElasticIndexesPath: elasticIndexesPath,
		Debug:              os.Getenv("APP_ENV") != "prod",
		LogLevel:           os.Getenv("LOG_LEVEL"),
		FrontendOrigin:     os.Getenv("FRONTEND_ORIGIN"),
		MigrationsPath:     migrationsPath,
		Origin:             origin,
		ProfileService:     "profile:" + "50050",
		AuthService:        "auth:" + "50050",
		FriendService:      "friend:" + "50050",
		MainService:        "main:" + "50050",
		PostgresURLProfile: postgresURLProfile,
		PostgresURLMain:    postgresURLMain,
		PostgresURLAuth:    postgresURLAuth,
		PostgresURLFriend:  postgresURLFrined,
	}
	return config
}
