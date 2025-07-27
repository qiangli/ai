package llm

import (
	"os"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/qiangli/ai/internal/proxy/llm/handler"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

func Start(cfg *api.AppConfig) {
	handler.ApiSecret = cfg.Hub.LLMProxySecret
	handler.ApiKey = cfg.Hub.LLMProxyApiKey

	addr := config.Option{
		F: func(o *config.Options) {
			o.Addr = cfg.Hub.LLMProxyAddress
		},
	}

	s := server.Default(addr)
	s.SetCustomSignalWaiter(func(err chan error) error {
		msg := <-err
		log.Debugf("LLM proxy exiting %v\n", msg)
		os.Exit(0)
		return nil
	})

	s.GET("/", handler.Ping)
	s.GET("/ping", handler.Ping)

	s.GET("/v1/models", handler.OpenAiModels)
	s.POST("/v1/chat/completions", handler.OpenAiV1ChatCompletions)

	s.GET("/openai/v1/models", handler.OpenAiModels)
	s.POST("/openai/v1/chat/completions", handler.OpenAiV1ChatCompletions)

	log.Infof("LLM Proxy listening at %s...\n", cfg.Hub.LLMProxyAddress)

	s.Spin()
}
