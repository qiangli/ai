package memory

// import (
// 	"testing"
// )

// func TestKGManager(t *testing.T) {
// 	manager := NewKGManager()

// 	// Create entities
// 	entities := []*Entity{
// 		{Name: "John_Smith", EntityType: "person", Observations: []string{"Speaks fluent Spanish"}},
// 		{Name: "Anthropic", EntityType: "organization", Observations: []string{"AI research"}},
// 	}
// 	manager.CreateEntities(entities)

// 	// Create relations
// 	relations := []*Relation{
// 		{From: "John_Smith", To: "Anthropic", RelationType: "works_at"},
// 	}
// 	manager.CreateRelations(relations)

// 	// Add observations
// 	observations := []*Observation{
// 		{EntityName: "John_Smith", Contents: []string{"Graduated in 2019", "Prefers morning meetings"}},
// 	}
// 	manager.AddObservations(observations)

// 	// Read graph
// 	graph := manager.ReadGraph()
// 	t.Log("Entities:", graph.Entities)
// 	t.Log("Relations:", graph.Relations)

// 	// Search nodes
// 	results := manager.SearchNodes("John")
// 	t.Log("Search Results:", results)

// 	// Open nodes
// 	results = manager.OpenNodes([]string{"John_Smith", "Anthropic"})
// 	t.Log("Open results:", results)
// }
