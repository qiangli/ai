package db

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/alicebob/miniredis/v2/server"

	"github.com/qiangli/ai/internal/log"
)

func StartRedis(addr string) {
	m := miniredis.NewMiniRedis()

	err := m.StartAddr(addr)
	if err != nil {
		log.Errorf("failed to start redis: %v\n", err)
		return
	}
	defer m.Close()

	log.Infof("Redis listening on %s...\n", addr)

	var hook = func(peer *server.Peer, cmd string, params ...string) bool {
		log.Debugf("hook %s %s %v\n", peer.ClientName, cmd, params)
		return false
	}
	m.Server().SetPreHook(hook)

	select {}
}
