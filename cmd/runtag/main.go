package main

import (
	"fmt"
	"log"

	"github.com/qiangli/ai/swarm/atm/gitkit"
)

func main() {
	args := &gitkit.Args{Dir: "/private/tmp/gittest", TagName: "v1.0", Rev: "HEAD"}
	out, err := gitkit.RunGitTag(args)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Println(out)
}
