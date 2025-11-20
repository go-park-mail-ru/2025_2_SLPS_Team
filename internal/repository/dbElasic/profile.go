package dbElasic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/mozillazg/go-unidecode"
)

type ElasticProfileStore struct {
	client *elasticsearch.Client
	index  string
}

func NewElasticProfileStore(client *elasticsearch.Client, index string) *ElasticProfileStore {
	return &ElasticProfileStore{
		client: client,
		index:  index,
	}
}

type ProfileDocument struct {
	UserID           int    `json:"user_id"`
	FullName         string `json:"full_name"`
	FullNameTranslit string `json:"full_name_translit"`
}

func (e *ElasticProfileStore) CreateProfile(ctx context.Context, fullName string, userID int) error {
	translit := unidecode.Unidecode(fullName)
	doc := ProfileDocument{
		UserID:           userID,
		FullName:         fullName,
		FullNameTranslit: translit,
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	req := esapi.IndexRequest{
		Index:      e.index,
		DocumentID: strconv.Itoa(userID),
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error indexing profile: %s", res.String())
	}
	return nil
}

func (e *ElasticProfileStore) UpdateProfile(ctx context.Context, fullName string, userID int) error {
	translit := unidecode.Unidecode(fullName)
	updateDoc := map[string]interface{}{
		"doc": map[string]interface{}{
			"full_name":          fullName,
			"full_name_translit": translit,
		},
	}
	body, _ := json.Marshal(updateDoc)

	req := esapi.UpdateRequest{
		Index:      e.index,
		DocumentID: strconv.Itoa(userID),
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error updating profile: %s", res.String())
	}
	return nil
}

func (e *ElasticProfileStore) DeleteProfile(ctx context.Context, userID int) error {
	req := esapi.DeleteRequest{
		Index:      e.index,
		DocumentID: strconv.Itoa(userID),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return errors.New("profile not found")
	}
	if res.IsError() {
		return fmt.Errorf("error deleting profile: %s", res.String())
	}
	return nil
}

func (e *ElasticProfileStore) SearchProfileIDsByFullName(ctx context.Context, fullName string) ([]int, error) {
	qEn := unidecode.Unidecode(fullName)

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{
						"match": map[string]interface{}{
							"full_name": fullName,
						},
					},
					{
						"match": map[string]interface{}{
							"full_name_translit": fullName,
						},
					},
					{
						"match": map[string]interface{}{
							"full_name": qEn,
						},
					},
					{
						"match": map[string]interface{}{
							"full_name_translit": qEn,
						},
					},
				},
				"minimum_should_match": 1,
			},
		},
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	res, err := e.client.Search(
		e.client.Search.WithContext(ctx),
		e.client.Search.WithIndex(e.index),
		e.client.Search.WithBody(bytes.NewReader(body)),
		e.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error searching profiles: %s", res.String())
	}

	var r struct {
		Hits struct {
			Hits []struct {
				Source ProfileDocument `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}

	// Собираем уникальные user_id
	idsMap := make(map[int]struct{})
	for _, hit := range r.Hits.Hits {
		idsMap[hit.Source.UserID] = struct{}{}
	}

	ids := make([]int, 0, len(idsMap))
	for id := range idsMap {
		ids = append(ids, id)
	}

	return ids, nil
}
