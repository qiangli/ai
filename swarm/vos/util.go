package vos

import (
	"os"
	"sort"
	"strings"
)

func GetEnvVarNames() string {
	names := []string{}
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			names = append(names, pair[0])
		}
	}
	sort.Strings(names)
	return strings.Join(names, "\n")
}
