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

type ElasticCommunityStore struct {
	client *elasticsearch.Client
	index  string
}

func NewElasticCommunityStore(client *elasticsearch.Client, index string) domain.ElasticCommunityStore {
	return &ElasticCommunityStore{
		client: client,
		index:  index,
	}
}

type CommunityDocument struct {
	CommunityID           int    `json:"community_id"`
	CommunityName         string `json:"community_name"`
	CommunityNameTranslit string `json:"community_name_translit"`
}

func (e *ElasticCommunityStore) CreateCommunity(ctx context.Context, name string, communityID int) error {
	translit := unidecode.Unidecode(name)
	doc := CommunityDocument{
		CommunityID:           communityID,
		CommunityName:         name,
		CommunityNameTranslit: translit,
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	req := esapi.IndexRequest{
		Index:      e.index,
		DocumentID: strconv.Itoa(communityID),
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error indexing community: %s", res.String())
	}
	return nil
}

func (e *ElasticCommunityStore) UpdateCommunity(ctx context.Context, name string, communityID int) error {
	translit := unidecode.Unidecode(name)
	updateDoc := map[string]interface{}{
		"doc": map[string]interface{}{
			"community_name":          name,
			"community_name_translit": translit,
		},
	}
	body, _ := json.Marshal(updateDoc)

	req := esapi.UpdateRequest{
		Index:      e.index,
		DocumentID: strconv.Itoa(communityID),
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error updating community: %s", res.String())
	}
	return nil
}

func (e *ElasticCommunityStore) DeleteCommunity(ctx context.Context, communityID int) error {
	req := esapi.DeleteRequest{
		Index:      e.index,
		DocumentID: strconv.Itoa(communityID),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return errors.New("community not found")
	}
	if res.IsError() {
		return fmt.Errorf("error deleting community: %s", res.String())
	}
	return nil
}

func (e *ElasticCommunityStore) SearchCommunityIDsByName(ctx context.Context, name string, filterIDs []int, isTerms bool, limit, offset int) ([]int, error) {
	if isTerms && len(filterIDs) == 0 {
		return []int{}, nil
	}

	multiMatch := map[string]interface{}{
		"multi_match": map[string]interface{}{
			"query":     name,
			"fields":    []string{"community_name", "community_name_translit"},
			"fuzziness": "AUTO",
		},
	}

	boolQuery := map[string]interface{}{
		"must": []interface{}{multiMatch},
	}

	if len(filterIDs) > 0 {
		if isTerms {
			boolQuery["filter"] = []map[string]interface{}{
				{"terms": map[string]interface{}{"community_id": filterIDs}},
			}
		} else {
			boolQuery["must_not"] = []map[string]interface{}{
				{"terms": map[string]interface{}{"community_id": filterIDs}},
			}
		}
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": boolQuery,
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
				Source struct {
					CommunityID int `json:"community_id"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}

	ids := make([]int, 0, len(r.Hits.Hits))
	for _, hit := range r.Hits.Hits {
		ids = append(ids, hit.Source.CommunityID)
	}

	return ids, nil
}
