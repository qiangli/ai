package swarm

import (
	"github.com/qiangli/ai/swarm/api"
)

func (h *agentHandler) flowSequence(req *api.Request, resp *api.Response) error {
	// for _, task := range h.agent.Sub.Tasks {
	// 	//nreq := req.Clone()
	// 	//nreq.task.Name

	// }
	return nil
}

func (h *agentHandler) flowParallel(req *api.Request, resp *api.Response) error {
	return nil
}

func (h *agentHandler) flowChoice(req *api.Request, resp *api.Response) error {
	return nil
}

func (h *agentHandler) flowMap(req *api.Request, resp *api.Response) error {
	return nil
}
