package dbElasic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"project/domain"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/mozillazg/go-unidecode"
)

type ElasticProfileStore struct {
	client *elasticsearch.Client
	index  string
}

func NewElasticProfileStore(client *elasticsearch.Client, index string) domain.ElasticProfileStore {
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

func (e *ElasticProfileStore) SearchUserIDsByFullNameWithFilter(ctx context.Context, fullName string, filterIDs []int, isTerms bool, limit, offset int) ([]int, error) {
	qEn := unidecode.Unidecode(fullName)
	queries := []string{fullName, qEn}
	if isTerms && len(filterIDs) == 0 {
		return []int{}, nil
	}
	shouldClauses := []map[string]interface{}{}
	for _, q := range queries {
		shouldClauses = append(shouldClauses,
			map[string]interface{}{
				"multi_match": map[string]interface{}{
					"query":     q,
					"fields":    []string{"full_name", "full_name_translit"},
					"fuzziness": "AUTO",
					"type":      "best_fields",
				},
			},
		)
	}

	boolQuery := map[string]interface{}{
		"should":               shouldClauses,
		"minimum_should_match": 1,
	}

	if len(filterIDs) > 0 {
		if isTerms {
			boolQuery["filter"] = []map[string]interface{}{
				{"terms": map[string]interface{}{"user_id": filterIDs}},
			}
		} else {
			boolQuery["must_not"] = []map[string]interface{}{
				{"terms": map[string]interface{}{"user_id": filterIDs}},
			}
		}
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
	}

	body, _ := json.Marshal(query)
	res, err := e.client.Search(
		e.client.Search.WithContext(ctx),
		e.client.Search.WithIndex(e.index),
		e.client.Search.WithBody(bytes.NewReader(body)),
		e.client.Search.WithTrackTotalHits(true),
		e.client.Search.WithSize(limit),
		e.client.Search.WithFrom(offset),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

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

	ids := make([]int, 0, len(r.Hits.Hits))
	for _, hit := range r.Hits.Hits {
		ids = append(ids, hit.Source.UserID)
	}

	return ids, nil
}
