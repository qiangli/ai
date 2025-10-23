package memory

type KnowledgeGraph struct {
	Entities  map[string]*Entity
	Relations []*Relation
}

type Entity struct {
	Name         string   `json:"name"`
	EntityType   string   `json:"entity_type"`
	Observations []string `json:"observations"`
}

type Deletion struct {
	EntityName   string
	Observations []string
}

type Relation struct {
	From         string `json:"from"`
	To           string `json:"to"`
	RelationType string `json:"relation_type"`
}

type Observation struct {
	EntityName string   `json:"entity_name"`
	Contents   []string `json:"contents"`
}
