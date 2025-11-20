package main

import (
	"bytes"
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

func CreateProfileIndex(indexName string, config *config.Config) error {

	es := NewElastic(config)
	indexConfig := []byte(`
 {
  "settings": {
    "analysis": {
      "analyzer": {
        "name_analyzer": {
          "tokenizer": "standard",
          "filter": [
            "lowercase",
            "russian_morphology",
            "english_morphology",
            "asciifolding",
            "edge_ngram_filter",
            "name_phonetic"
          ]
        },
        "name_search_analyzer": {
          "tokenizer": "standard",
          "filter": [
            "lowercase",
            "russian_morphology",
            "english_morphology",
            "asciifolding",
            "name_phonetic"
          ]
        }
      },
      "filter": {
        "russian_morphology": { "type": "snowball", "language": "russian" },
        "english_morphology": { "type": "snowball", "language": "english" },
        "edge_ngram_filter": { "type": "edge_ngram", "min_gram": 2, "max_gram": 20 },
        "name_phonetic": { "type": "phonetic", "encoder": "metaphone", "languageset": ["russian","english"], "replace": false }
      }
    }
  },
  "mappings": {
    "properties": {
      "full_name": { "type": "text", "analyzer": "name_analyzer", "search_analyzer": "name_search_analyzer" },
      "full_name_translit": { "type": "text", "analyzer": "name_analyzer", "search_analyzer": "name_search_analyzer" },
      "user_id": { "type": "integer" }
    }
  }
}




`)

	existsResp, err := es.Indices.Exists([]string{indexName})
	if err != nil {
		return err
	}
	defer existsResp.Body.Close()

	if existsResp.StatusCode == 200 {
		fmt.Println("Index already exists:", indexName)
		return nil
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
		return fmt.Errorf("error creating index: %s", body)
	}

	fmt.Println("Index created:", indexName)
	return nil
}

func main() {
	config := config.NewConfig()
	if config.Debug {
		log.Println("Debug mode enabled")
	}

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

		if err := CreateProfileIndex("profile", config); err != nil {
			log.Fatalf("Error applying migrations: %v", err)
		}
		log.Println("Migrations applied successfully.")
	}
}
