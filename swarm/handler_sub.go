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

func (h *agentHandler) flowCondition(req *api.Request, resp *api.Response) error {
	return nil
}

func (h *agentHandler) flowLoop(req *api.Request, resp *api.Response) error {
	return nil
}
