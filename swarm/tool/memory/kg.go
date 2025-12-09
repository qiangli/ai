package memory

// Entity represents a knowledge graph node with observations.
type Entity struct {
	Name         string   `json:"name"`
	EntityType   string   `json:"entityType"`
	Observations []string `json:"observations"`
}

// Relation represents a directed edge between two entities.
type Relation struct {
	From         string `json:"from"`
	To           string `json:"to"`
	RelationType string `json:"relationType"`
}

// Observation contains facts about an entity.
type Observation struct {
	EntityName string   `json:"entityName"`
	Contents   []string `json:"contents"`

	Observations []string `json:"observations,omitempty"` // Used for deletion operations
}

// KnowledgeGraph represents the complete graph structure.
type KnowledgeGraph struct {
	Entities  []Entity   `json:"entities"`
	Relations []Relation `json:"relations"`
}

// type KnowledgeGraph struct {
// 	Entities  map[string]*Entity
// 	Relations []*Relation
// }

// type Entity struct {
// 	Name         string   `json:"name"`
// 	EntityType   string   `json:"entity_type"`
// 	Observations []string `json:"observations"`
// }

// type Deletion struct {
// 	EntityName   string
// 	Observations []string
// }

// type Relation struct {
// 	From         string `json:"from"`
// 	To           string `json:"to"`
// 	RelationType string `json:"relation_type"`
// }

// type Observation struct {
// 	EntityName string   `json:"entity_name"`
// 	Contents   []string `json:"contents"`
// }

// // CreateEntitiesArgs defines the create entities tool parameters.
// type CreateEntitiesArgs struct {
// 	Entities []Entity `json:"entities" mcp:"entities to create"`
// }

// // CreateEntitiesResult returns newly created entities.
// type CreateEntitiesResult struct {
// 	Entities []Entity `json:"entities"`
// }

// // CreateRelationsArgs defines the create relations tool parameters.
// type CreateRelationsArgs struct {
// 	Relations []Relation `json:"relations" mcp:"relations to create"`
// }

// // CreateRelationsResult returns newly created relations.
// type CreateRelationsResult struct {
// 	Relations []Relation `json:"relations"`
// }

// // AddObservationsArgs defines the add observations tool parameters.
// type AddObservationsArgs struct {
// 	Observations []Observation `json:"observations" mcp:"observations to add"`
// }

// // AddObservationsResult returns newly added observations.
// type AddObservationsResult struct {
// 	Observations []Observation `json:"observations"`
// }

// // DeleteEntitiesArgs defines the delete entities tool parameters.
// type DeleteEntitiesArgs struct {
// 	EntityNames []string `json:"entityNames" mcp:"entities to delete"`
// }

// // DeleteObservationsArgs defines the delete observations tool parameters.
// type DeleteObservationsArgs struct {
// 	Deletions []Observation `json:"deletions" mcp:"obeservations to delete"`
// }

// // DeleteRelationsArgs defines the delete relations tool parameters.
// type DeleteRelationsArgs struct {
// 	Relations []Relation `json:"relations" mcp:"relations to delete"`
// }

// // SearchNodesArgs defines the search nodes tool parameters.
// type SearchNodesArgs struct {
// 	Query string `json:"query" mcp:"query string"`
// }

// // OpenNodesArgs defines the open nodes tool parameters.
// type OpenNodesArgs struct {
// 	Names []string `json:"names" mcp:"names of nodes to open"`
// }
