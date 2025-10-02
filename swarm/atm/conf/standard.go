package conf

import (
	"fmt"

	// log "github.com/sirupsen/logrus"

	"github.com/qiangli/ai/swarm/api"
	assetData "github.com/qiangli/ai/swarm/atm/resource"
)

var standardAgentNames = []string{"agent", "ask", "web"}

var (
	standardAgents map[string]*api.AgentsConfig
	standardModels map[string]*api.ModelsConfig
	standardTools  map[string]*api.ToolsConfig
)

// load builtin defaults
func InitStandard() error {
	// log.Infof("standard agent init...")
	if v, err := loadStandardAgentsConfig(); err != nil {
		return err
	} else {
		standardAgents = v
	}

	// log.Infof("standard models init...")
	if v, err := loadStandardModels(); err != nil {
		return err
	} else {
		standardModels = v
	}

	// log.Infof("standard tool init...")
	if v, err := loadStandardTools(); err != nil {
		return err
	} else {
		standardTools = v
	}
	return nil
}

func loadStandardAgentsConfig() (map[string]*api.AgentsConfig, error) {
	var agents = make(map[string]*api.AgentsConfig)

	groups := make(map[string]*api.AgentsConfig)
	rs := assetData.NewStandardStore()
	if err := LoadAgentsAsset(rs, "agents", groups); err != nil {
		return nil, err
	}

	for name, ac := range groups {
		// log.Debugf("Registering agent: %s with %d configurations\n", name, len(ac.Agents))

		if len(ac.Agents) == 0 {
			return nil, fmt.Errorf("No agent configurations found for: %s\n", name)
		}

		// Register the agent configurations
		for _, agent := range ac.Agents {
			if _, exists := agents[agent.Name]; exists {
				return nil, fmt.Errorf("Duplicate agent name found: %s, skipping registration\n", agent.Name)
			}
			// Register the agents configuration
			agents[agent.Name] = ac

			// log.Debugf("Registered agent: %s\n", agent.Name)
			if ac.MaxTurns == 0 {
				ac.MaxTurns = defaultMaxTurns
			}
			if ac.MaxTime == 0 {
				ac.MaxTime = defaultMaxTime
			}
			// upper limit
			ac.MaxTurns = min(ac.MaxTurns, maxTurnsLimit)
			ac.MaxTime = min(ac.MaxTime, maxTimeLimit)
		}
	}

	if len(agents) == 0 {
		return nil, fmt.Errorf("no agent configurations found in default agents")
	}

	// log.Debugf("Initialized %d agent configurations", len(agents))

	return agents, nil
}

func loadStandardModels() (map[string]*api.ModelsConfig, error) {
	// var modelsMap = make(map[string]api.ModelAlias)

	modelsCfg := make(map[string]*api.ModelsConfig)
	rs := assetData.NewStandardStore()
	if err := loadModelsAsset(rs, "models", modelsCfg); err != nil {
		return nil, err
	}
	return modelsCfg, nil
	// for alias, mc := range modelsCfg {
	// 	var modelMap = make(map[string]*api.Model)
	// 	for k, v := range mc.Models {
	// 		modelMap[k] = &api.Model{
	// 			Provider: v.Provider,
	// 			Model:    v.Model,
	// 			BaseUrl:  v.BaseUrl,
	// 			// ApiKey:   "",
	// 			//
	// 			Config: mc,
	// 		}
	// 	}
	// 	modelsMap[alias] = modelMap
	// }

	// return modelsMap, nil
}

func loadStandardTools() (map[string]*api.ToolsConfig, error) {
	// var toolMap = make(map[string]*api.ToolFunc)

	kits := make(map[string]*api.ToolsConfig)
	rs := assetData.NewStandardStore()
	if err := loadToolsAsset(rs, "tools", kits); err != nil {
		return nil, err
	}

	return kits, nil

	// toolMap, err := loadTools(kits)
	// if err != nil {
	// 	return nil, err
	// }

	// var tools = make([]*api.ToolFunc, 0)
	// for _, v := range toolMap {
	// 	tools = append(tools, v)
	// }
	// return tools, nil
}
