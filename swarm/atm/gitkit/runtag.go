package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/qiangli/ai/swarm/atm/gitkit"
)

func main() {
	args := &amp;gitkit.Args{
		Dir:     &quot;/private/tmp/gittest&quot;,
		TagName: &quot;v1.0&quot;,
		Rev:     &quot;HEAD&quot;,
	}
	out, err := gitkit.RunGitTag(args)
	if err != nil {
		fmt.Printf(&quot;Error: %v\n&quot;, err)
		os.Exit(1)
	}
	bs, _ := json.MarshalIndent(out, &quot;&quot;, &quot;  &quot;)
	fmt.Println(&quot;Output:&quot;, string(bs))
}