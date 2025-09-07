package xterm

import (
	"context"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/xterm/localcommand"
	"github.com/qiangli/ai/internal/xterm/server"
	"github.com/qiangli/ai/swarm/api"
)

// https://github.com/sorenisanerd/gotty.git
func Start(cfg *api.AppConfig) error {
	// if !cfg.Hub.Terminal {
	// 	log.Debugf("Hub terminal service is disabled\n")
	// 	return nil
	// }

	address := ":58082"

	appOptions := &server.Options{
		PermitWrite: true,
		Address:     address,
		Path:        "terminal",
	}

	backendOptions := &localcommand.Options{}
	factory, err := localcommand.NewFactory("ai", []string{"-i"}, backendOptions)
	if err != nil {
		return err
	}

	srv, err := server.New(factory, appOptions)
	if err != nil {
		return err
	}

	ctx := context.Background()

	log.Debugln("Web terminal is starting ...\n")

	errs := make(chan error, 1)
	go func() {
		errs <- srv.Run(ctx)
	}()

	if err := <-errs; err != nil {
		return err
	}

	return nil
}
