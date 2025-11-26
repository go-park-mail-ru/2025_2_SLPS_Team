package dbconn

import (
	"database/sql"
	"log"
	"project/config"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gomodule/redigo/redis"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgres(dataSourceName string) *sql.DB {
	db, err := sql.Open("pgx", dataSourceName)
	if err != nil {
		log.Fatalf("ошибка подключения к БД: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("ошибка ping БД: %v", err)
	}

	return db
}

func NewRedisPool(dataSourceName string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:   10,
		MaxActive: 50, // 0 = без лимита
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(dataSourceName)
			if err != nil {
				log.Fatalf("Can't connect to Redis: %v", err)
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		IdleTimeout: 240 * time.Second,
	}
}

func NewElastic(config *config.Config) *elasticsearch.Client {
	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://elasticsearch:" + config.ElasticPort,
		},
		Username: config.ElasticUser,
		Password: config.ElasticPassword,
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Ошибка создания клиента: %s", err)
	}

	res, err := es.Info()
	if err != nil {
		log.Fatalf("Ошибка подключения: %s", err)
	}
	defer res.Body.Close()
	log.Println("Elasticsearch подключен:", res.Status())
	return es
}
