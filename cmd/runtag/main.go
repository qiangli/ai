package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/qiangli/ai/swarm/atm/gitkit"
)

func main() {
	args := &gitkit.Args{
		Dir:     "/private/tmp/gittest",
		TagName: "v1.0",
		Rev:     "HEAD",
	}
	out, err := gitkit.RunGitTag(args)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	bs, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println("Output:", string(bs))
}
