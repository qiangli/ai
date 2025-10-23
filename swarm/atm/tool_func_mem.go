package atm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/tool/memory"
)

func anyToStruct(input any, obj any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal to JSON: %v", err)
	}

	if err := json.Unmarshal(data, obj); err != nil {
		return fmt.Errorf("failed to unmarshal: %v", err)
	}
	return nil
}

func toJsonString(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *FuncKit) CreateEntities(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	data, ok := args["entities"]
	if !ok {
		return "", fmt.Errorf("missing arguments: entities")
	}

	var entities []*memory.Entity
	if err := anyToStruct(data, &entities); err != nil {
		return "", err
	}

	r.kgm.CreateEntities(entities)

	return "success", nil
}

func (r *FuncKit) CreateRelations(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	data, ok := args["relations"]
	if !ok {
		return "", fmt.Errorf("missing arguments: relations")
	}

	var relations []*memory.Relation
	if err := anyToStruct(data, &relations); err != nil {
		return "", err
	}

	r.kgm.CreateRelations(relations)

	return "success", nil
}

func (r *FuncKit) AddObservations(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	data, ok := args["observations"]
	if !ok {
		return "", fmt.Errorf("missing arguments: observations")
	}

	var observations []*memory.Observation
	if err := anyToStruct(data, &observations); err != nil {
		return "", err
	}

	r.kgm.AddObservations(observations)

	return "success", nil
}

func (r *FuncKit) DeleteEntities(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	entityNames, ok := args["entity_names"]
	if !ok {
		return "", fmt.Errorf("missing arguments: entity_names")
	}

	if v, ok := entityNames.([]string); ok {
		r.kgm.DeleteEntities(v)
		return "success", nil
	}
	return "", fmt.Errorf("invalid arguments: %v. expectded array of strings.", entityNames)
}

func (r *FuncKit) DeleteObservations(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	data, ok := args["deletions"]
	if !ok {
		return "", fmt.Errorf("missing arguments: deletions")
	}

	var deletions []*memory.Deletion
	if err := anyToStruct(data, &deletions); err != nil {
		return "", err
	}

	r.kgm.DeleteObservations(deletions)

	return "success", nil
}

func (r *FuncKit) DeleteRelations(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	data, ok := args["relations"]
	if !ok {
		return "", fmt.Errorf("missing arguments: relations")
	}

	var relations []*memory.Relation
	if err := anyToStruct(data, &relations); err != nil {
		return "", err
	}

	r.kgm.DeleteRelations(relations)

	return "success", nil
}

func (r *FuncKit) ReadGraph(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	gd := r.kgm.ReadGraph()
	return toJsonString(gd)
}

func (r *FuncKit) SearchNodes(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	query, ok := args["query"]
	if !ok {
		return "", fmt.Errorf("missing argument: query")
	}

	results := r.kgm.SearchNodes(query.(string))
	return toJsonString(results)
}

func (r *FuncKit) OpenNodes(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	names, ok := args["names"]
	if !ok {
		return "", fmt.Errorf("missing argument: names")
	}

	if v, ok := names.([]string); ok {
		d := r.kgm.OpenNodes(v)
		return toJsonString(d)
	}
	return "", fmt.Errorf("invalide arguments: %v. expected array of strings.", names)
}
