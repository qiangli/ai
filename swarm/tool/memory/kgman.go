package memory

import (
	"strings"
)

type KGManager struct {
	Graph *KnowledgeGraph
}

func NewKGManager() *KGManager {
	return &KGManager{
		Graph: &KnowledgeGraph{
			Entities:  make(map[string]*Entity),
			Relations: []*Relation{},
		},
	}
}

func (kg *KGManager) CreateEntities(entities []*Entity) {
	for _, entity := range entities {
		if _, exists := kg.Graph.Entities[entity.Name]; !exists {
			kg.Graph.Entities[entity.Name] = entity
		}
	}
}

func (kg *KGManager) CreateRelations(relations []*Relation) {
	for _, relation := range relations {
		if !kg.relationExists(relation) {
			kg.Graph.Relations = append(kg.Graph.Relations, relation)
		}
	}
}

func (kg *KGManager) AddObservations(observations []*Observation) map[string][]string {
	addedObservations := make(map[string][]string)
	for _, obs := range observations {
		if entity, exists := kg.Graph.Entities[obs.EntityName]; exists {
			for _, content := range obs.Contents {
				if !contains(entity.Observations, content) {
					entity.Observations = append(entity.Observations, content)
					addedObservations[obs.EntityName] = append(addedObservations[obs.EntityName], content)
				}
			}
		}
	}
	return addedObservations
}

func (kg *KGManager) DeleteEntities(entityNames []string) {
	for _, name := range entityNames {
		delete(kg.Graph.Entities, name)
		kg.deleteRelationsByEntity(name)
	}
}

func (kg *KGManager) DeleteObservations(deletions []*Deletion) {
	for _, del := range deletions {
		if entity, exists := kg.Graph.Entities[del.EntityName]; exists {
			for _, obs := range del.Observations {
				entity.Observations = remove(entity.Observations, obs)
			}
		}
	}
}

func (kg *KGManager) DeleteRelations(relations []*Relation) {
	for _, rel := range relations {
		kg.Graph.Relations = filterRelations(kg.Graph.Relations, rel)
	}
}

func (kg *KGManager) ReadGraph() *KnowledgeGraph {
	return kg.Graph
}

func (kg *KGManager) SearchNodes(query string) *KnowledgeGraph {
	query = strings.ToLower(query)
	var results = make(map[string]*Entity)
	for name, entity := range kg.Graph.Entities {
		if strings.Contains(strings.ToLower(entity.Name), query) ||
			strings.Contains(strings.ToLower(entity.EntityType), query) ||
			containsCaseInsensitive(entity.Observations, query) {
			results[name] = entity
		}
	}

	entityNames := make(map[string]struct{})
	for _, entity := range results {
		entityNames[entity.Name] = struct{}{}
	}

	var filteredRelations []*Relation
	for _, relation := range kg.Graph.Relations {
		if _, fromExists := entityNames[relation.From]; fromExists {
			if _, toExists := entityNames[relation.To]; toExists {
				filteredRelations = append(filteredRelations, relation)
			}
		}
	}

	return &KnowledgeGraph{
		Entities:  results,
		Relations: filteredRelations,
	}
}

func (kg *KGManager) OpenNodes(names []string) *KnowledgeGraph {
	entityNames := make(map[string]struct{})
	for _, name := range names {
		entityNames[name] = struct{}{}
	}

	var results = make(map[string]*Entity)
	for name, entity := range kg.Graph.Entities {
		if _, exists := entityNames[entity.Name]; exists {
			results[name] = entity
		}
	}

	var filteredRelations []*Relation
	for _, relation := range kg.Graph.Relations {
		if _, fromExists := entityNames[relation.From]; fromExists {
			if _, toExists := entityNames[relation.To]; toExists {
				filteredRelations = append(filteredRelations, relation)
			}
		}
	}

	return &KnowledgeGraph{
		Entities:  results,
		Relations: filteredRelations,
	}
}

func (kg *KGManager) relationExists(relation *Relation) bool {
	for _, rel := range kg.Graph.Relations {
		if rel.From == relation.From && rel.To == relation.To && rel.RelationType == relation.RelationType {
			return true
		}
	}
	return false
}

func (kg *KGManager) deleteRelationsByEntity(entityName string) {
	var updatedRelations []*Relation
	for _, rel := range kg.Graph.Relations {
		if rel.From != entityName && rel.To != entityName {
			updatedRelations = append(updatedRelations, rel)
		}
	}
	kg.Graph.Relations = updatedRelations
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsCaseInsensitive(observations []string, query string) bool {
	for _, observation := range observations {
		if strings.Contains(strings.ToLower(observation), query) {
			return true
		}
	}
	return false
}

func remove(slice []string, item string) []string {
	var result []string
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

func filterRelations(relations []*Relation, target *Relation) []*Relation {
	var result []*Relation
	for _, rel := range relations {
		if !(rel.From == target.From && rel.To == target.To && rel.RelationType == target.RelationType) {
			result = append(result, rel)
		}
	}
	return result
}
