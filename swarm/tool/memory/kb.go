// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package memory

import (
	// "context"
	"encoding/json"
	"fmt"
	// "os"
	"slices"
	"strings"
	// "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qiangli/shell/tool/sh/vfs"
)

// store provides persistence interface for knowledge base data.
type store interface {
	Read() ([]byte, error)
	Write(data []byte) error
}

// memoryStore implements in-memory storage that doesn't persist across restarts.
type memoryStore struct {
	data []byte
}

// Read returns the in-memory data.
func (ms *memoryStore) Read() ([]byte, error) {
	return ms.data, nil
}

// Write stores data in memory.
func (ms *memoryStore) Write(data []byte) error {
	ms.data = data
	return nil
}

// fileStore implements file-based storage for persistent knowledge base.
type fileStore struct {
	path string

	store vfs.FileStore
}

// Read loads data from file, returning empty slice if file doesn't exist.
func (fs *fileStore) Read() ([]byte, error) {
	data, err := fs.store.ReadFile(fs.path, nil)
	if err != nil {
		// if os.IsNotExist(err) {
		// 	return nil, nil
		// }
		return nil, fmt.Errorf("failed to read file %s: %w", fs.path, err)
	}
	return data, nil
}

// Write saves data to file with 0600 permissions.
func (fs *fileStore) Write(data []byte) error {
	if err := fs.store.WriteFile(fs.path, data); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fs.path, err)
	}
	return nil
}

// KnowledgeBase manages entities and relations with persistent storage.
type KnowledgeBase struct {
	s store
}

func NewKnowlegeBase(path string) *KnowledgeBase {
	kbStore := &fileStore{path: path}
	kb := KnowledgeBase{s: kbStore}
	return &kb
}

// kbItem represents a single item in persistent storage (entity or relation).
type kbItem struct {
	Type string `json:"type"`

	// Entity fields (when Type == "entity")
	Name         string   `json:"name,omitempty"`
	EntityType   string   `json:"entityType,omitempty"`
	Observations []string `json:"observations,omitempty"`

	// Relation fields (when Type == "relation")
	From         string `json:"from,omitempty"`
	To           string `json:"to,omitempty"`
	RelationType string `json:"relationType,omitempty"`
}

// loadGraph deserializes the knowledge graph from storage.
func (k KnowledgeBase) loadGraph() (KnowledgeGraph, error) {
	graph := KnowledgeGraph{}

	if k.s == nil {
		return graph, fmt.Errorf("knowlege store has not been initialized")
	}
	data, err := k.s.Read()
	if err != nil {
		return graph, fmt.Errorf("failed to read from store: %w", err)
	}

	if len(data) == 0 {
		return graph, nil
	}

	var items []kbItem
	if err := json.Unmarshal(data, &items); err != nil {
		return graph, fmt.Errorf("failed to unmarshal from store: %w", err)
	}

	for _, item := range items {
		switch item.Type {
		case "entity":
			graph.Entities = append(graph.Entities, Entity{
				Name:         item.Name,
				EntityType:   item.EntityType,
				Observations: item.Observations,
			})
		case "relation":
			graph.Relations = append(graph.Relations, Relation{
				From:         item.From,
				To:           item.To,
				RelationType: item.RelationType,
			})
		}
	}

	return graph, nil
}

// saveGraph serializes and persists the knowledge graph to storage.
func (k KnowledgeBase) saveGraph(graph KnowledgeGraph) error {
	items := make([]kbItem, 0, len(graph.Entities)+len(graph.Relations))

	for _, entity := range graph.Entities {
		items = append(items, kbItem{
			Type:         "entity",
			Name:         entity.Name,
			EntityType:   entity.EntityType,
			Observations: entity.Observations,
		})
	}

	for _, relation := range graph.Relations {
		items = append(items, kbItem{
			Type:         "relation",
			From:         relation.From,
			To:           relation.To,
			RelationType: relation.RelationType,
		})
	}

	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}

	if err := k.s.Write(itemsJSON); err != nil {
		return fmt.Errorf("failed to write to store: %w", err)
	}
	return nil
}

func (kb KnowledgeBase) ReadGraph() (*KnowledgeGraph, error) {
	graph, err := kb.loadGraph()
	if err != nil {
		return nil, err
	}
	return &graph, nil
}

// CreateEntities adds new entities to the graph, skipping duplicates by name.
// It returns the new entities that were actually added.
func (k KnowledgeBase) CreateEntities(entities []Entity) ([]Entity, error) {
	graph, err := k.loadGraph()
	if err != nil {
		return nil, err
	}

	var newEntities []Entity
	for _, entity := range entities {
		if !slices.ContainsFunc(graph.Entities, func(e Entity) bool { return e.Name == entity.Name }) {
			newEntities = append(newEntities, entity)
			graph.Entities = append(graph.Entities, entity)
		}
	}

	if err := k.saveGraph(graph); err != nil {
		return nil, err
	}

	return newEntities, nil
}

// CreateRelations adds new relations to the graph, skipping exact duplicates.
// It returns the new relations that were actually added.
func (k KnowledgeBase) CreateRelations(relations []Relation) ([]Relation, error) {
	graph, err := k.loadGraph()
	if err != nil {
		return nil, err
	}

	var newRelations []Relation
	for _, relation := range relations {
		exists := slices.ContainsFunc(graph.Relations, func(r Relation) bool {
			return r.From == relation.From &&
				r.To == relation.To &&
				r.RelationType == relation.RelationType
		})
		if !exists {
			newRelations = append(newRelations, relation)
			graph.Relations = append(graph.Relations, relation)
		}
	}

	if err := k.saveGraph(graph); err != nil {
		return nil, err
	}

	return newRelations, nil
}

// AddObservations appends new observations to existing entities.
// It returns the new observations that were actually added.
func (k KnowledgeBase) AddObservations(observations []Observation) ([]Observation, error) {
	graph, err := k.loadGraph()
	if err != nil {
		return nil, err
	}

	var results []Observation

	for _, obs := range observations {
		entityIndex := slices.IndexFunc(graph.Entities, func(e Entity) bool { return e.Name == obs.EntityName })
		if entityIndex == -1 {
			return nil, fmt.Errorf("entity with name %s not found", obs.EntityName)
		}

		var newObservations []string
		for _, content := range obs.Contents {
			if !slices.Contains(graph.Entities[entityIndex].Observations, content) {
				newObservations = append(newObservations, content)
				graph.Entities[entityIndex].Observations = append(graph.Entities[entityIndex].Observations, content)
			}
		}

		results = append(results, Observation{
			EntityName: obs.EntityName,
			Contents:   newObservations,
		})
	}

	if err := k.saveGraph(graph); err != nil {
		return nil, err
	}

	return results, nil
}

// DeleteEntities removes entities and their associated relations.
func (k KnowledgeBase) DeleteEntities(entityNames []string) error {
	graph, err := k.loadGraph()
	if err != nil {
		return err
	}

	// Create map for quick lookup
	entitiesToDelete := make(map[string]bool)
	for _, name := range entityNames {
		entitiesToDelete[name] = true
	}

	// Filter entities using slices.DeleteFunc
	graph.Entities = slices.DeleteFunc(graph.Entities, func(entity Entity) bool {
		return entitiesToDelete[entity.Name]
	})

	// Filter relations using slices.DeleteFunc
	graph.Relations = slices.DeleteFunc(graph.Relations, func(relation Relation) bool {
		return entitiesToDelete[relation.From] || entitiesToDelete[relation.To]
	})

	return k.saveGraph(graph)
}

// DeleteObservations removes specific observations from entities.
func (k KnowledgeBase) DeleteObservations(deletions []Observation) error {
	graph, err := k.loadGraph()
	if err != nil {
		return err
	}

	for _, deletion := range deletions {
		entityIndex := slices.IndexFunc(graph.Entities, func(e Entity) bool {
			return e.Name == deletion.EntityName
		})
		if entityIndex == -1 {
			continue
		}

		// Create a map for quick lookup
		observationsToDelete := make(map[string]bool)
		for _, observation := range deletion.Observations {
			observationsToDelete[observation] = true
		}

		// Filter observations using slices.DeleteFunc
		graph.Entities[entityIndex].Observations = slices.DeleteFunc(graph.Entities[entityIndex].Observations, func(observation string) bool {
			return observationsToDelete[observation]
		})
	}

	return k.saveGraph(graph)
}

// DeleteRelations removes specific relations from the graph.
func (k KnowledgeBase) DeleteRelations(relations []Relation) error {
	graph, err := k.loadGraph()
	if err != nil {
		return err
	}

	// Filter relations using slices.DeleteFunc and slices.ContainsFunc
	graph.Relations = slices.DeleteFunc(graph.Relations, func(existingRelation Relation) bool {
		return slices.ContainsFunc(relations, func(relationToDelete Relation) bool {
			return existingRelation.From == relationToDelete.From &&
				existingRelation.To == relationToDelete.To &&
				existingRelation.RelationType == relationToDelete.RelationType
		})
	})
	return k.saveGraph(graph)
}

// SearchNodes filters entities and relations matching the query string.
func (k KnowledgeBase) SearchNodes(query string) (KnowledgeGraph, error) {
	graph, err := k.loadGraph()
	if err != nil {
		return KnowledgeGraph{}, err
	}

	queryLower := strings.ToLower(query)
	var filteredEntities []Entity

	// Filter entities
	for _, entity := range graph.Entities {
		if strings.Contains(strings.ToLower(entity.Name), queryLower) ||
			strings.Contains(strings.ToLower(entity.EntityType), queryLower) {
			filteredEntities = append(filteredEntities, entity)
			continue
		}

		// Check observations
		for _, observation := range entity.Observations {
			if strings.Contains(strings.ToLower(observation), queryLower) {
				filteredEntities = append(filteredEntities, entity)
				break
			}
		}
	}

	// Create map for quick entity lookup
	filteredEntityNames := make(map[string]bool)
	for _, entity := range filteredEntities {
		filteredEntityNames[entity.Name] = true
	}

	// Filter relations
	var filteredRelations []Relation
	for _, relation := range graph.Relations {
		if filteredEntityNames[relation.From] && filteredEntityNames[relation.To] {
			filteredRelations = append(filteredRelations, relation)
		}
	}

	return KnowledgeGraph{
		Entities:  filteredEntities,
		Relations: filteredRelations,
	}, nil
}

// OpenNodes returns entities with specified names and their interconnecting relations.
func (k KnowledgeBase) OpenNodes(names []string) (KnowledgeGraph, error) {
	graph, err := k.loadGraph()
	if err != nil {
		return KnowledgeGraph{}, err
	}

	// Create map for quick name lookup
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	// Filter entities
	var filteredEntities []Entity
	for _, entity := range graph.Entities {
		if nameSet[entity.Name] {
			filteredEntities = append(filteredEntities, entity)
		}
	}

	// Create map for quick entity lookup
	filteredEntityNames := make(map[string]bool)
	for _, entity := range filteredEntities {
		filteredEntityNames[entity.Name] = true
	}

	// Filter relations
	var filteredRelations []Relation
	for _, relation := range graph.Relations {
		if filteredEntityNames[relation.From] && filteredEntityNames[relation.To] {
			filteredRelations = append(filteredRelations, relation)
		}
	}

	return KnowledgeGraph{
		Entities:  filteredEntities,
		Relations: filteredRelations,
	}, nil
}

// func (k KnowledgeBase) CreateEntities(ctx context.Context, args CreateEntitiesArgs) (CreateEntitiesResult, error) {
// 	entities, err := k.createEntities(args)
// 	if err != nil {
// 		return CreateEntitiesResult{}, err
// 	}
// 	return CreateEntitiesResult{Entities: entities}, nil
// }

// func (k KnowledgeBase) CreateRelations(ctx context.Context, args CreateRelationsArgs) (CreateRelationsResult, error) {

// 	relations, err := k.createRelations(args.Relations)
// 	if err != nil {
// 		return CreateRelationsResult{}, err
// 	}

// 	return CreateRelationsResult{Relations: relations}, nil
// }

// func (k KnowledgeBase) AddObservations(ctx context.Context, args AddObservationsArgs) (AddObservationsResult, error) {

// 	observations, err := k.addObservations(args.Observations)
// 	if err != nil {
// 		return AddObservationsResult{}, err
// 	}

// 	return AddObservationsResult{
// 		Observations: observations,
// 	}, nil
// }

// func (k KnowledgeBase) DeleteEntities(ctx context.Context, args DeleteEntitiesArgs) (any, error) {

// 	err := k.deleteEntities(args.EntityNames)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return nil, nil
// }

// func (k KnowledgeBase) DeleteObservations(ctx context.Context, args DeleteObservationsArgs) (any, error) {

// 	err := k.deleteObservations(args.Deletions)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return nil, nil
// }

// func (k KnowledgeBase) DeleteRelations(ctx context.Context, args DeleteRelationsArgs) (struct{}, error) {
// 	err := k.deleteRelations(args.Relations)
// 	if err != nil {
// 		return struct{}{}, err
// 	}
// 	return struct{}{}, nil
// }

// func (k KnowledgeBase) ReadGraph(ctx context.Context, args any) (KnowledgeGraph, error) {

// 	graph, err := k.loadGraph()
// 	if err != nil {
// 		return KnowledgeGraph{}, err
// 	}

// 	return graph, nil
// }

// func (k KnowledgeBase) SearchNodes(ctx context.Context, args SearchNodesArgs) (KnowledgeGraph, error) {

// 	graph, err := k.searchNodes(args.Query)
// 	if err != nil {
// 		return KnowledgeGraph{}, err
// 	}
// 	return graph, nil
// }

// func (k KnowledgeBase) OpenNodes(ctx context.Context, args OpenNodesArgs) (KnowledgeGraph, error) {
// 	graph, err := k.openNodes(args.Names)
// 	if err != nil {
// 		return KnowledgeGraph{}, err
// 	}

// 	return graph, nil
// }
