package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"project/config"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

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

func LoadIndexConfigs(path string) (map[string][]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for indexName, cfg := range raw {
		result[indexName] = cfg
	}

	return result, nil
}

func CreateIndexes(config *config.Config) error {
	es := NewElastic(config)

	indexConfigs, err := LoadIndexConfigs(config.ElasticIndexesPath)
	if err != nil {
		return err
	}

	for indexName, indexConfig := range indexConfigs {

		existsResp, err := es.Indices.Exists([]string{indexName})
		if err != nil {
			return err
		}
		existsResp.Body.Close()

		if existsResp.StatusCode == 200 {
			fmt.Println("Index already exists:", indexName)
			continue
		}

		res, err := es.Indices.Create(
			indexName,
			es.Indices.Create.WithBody(bytes.NewReader(indexConfig)),
		)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if res.IsError() {
			body, _ := io.ReadAll(res.Body)
			return fmt.Errorf("error creating index %s: %s", indexName, body)
		}

		fmt.Println("Index created:", indexName)
	}

	return nil
}

func main() {
	config := config.NewConfig()
	if config.Debug {
		log.Println("Debug mode enabled")
	}
	log.Println(config.MigrationsPath)
	log.Println(config.PostgresURL)
	m, err := migrate.New(
		config.MigrationsPath,
		config.PostgresURL,
	)
	if err != nil {
		log.Panicf("Error initializing migrations: %v", err)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "down":
			if err = m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
				log.Fatalf("Error rolling back migrations: %v", err)
			}
			log.Println("Migrations rolled back successfully.")

		default:
			if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
				log.Fatalf("Error applying migrations: %v", err)
			}
			log.Println("Migrations applied successfully.")
		}

	} else {
		if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("Error applying migrations: %v", err)
		}
		if err := CreateIndexes(config); err != nil {
			log.Fatalf("Error applying migrations: %v", err)
		}
		log.Println("Migrations applied successfully.")
	}
}
