#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script

set -ue

# # // | fromJson | toPrettyJson
# template='data:,
# >>>>>>>>> Instruction/Prompt

# {{.prompt}}

# >>>>>>>>> Context/History

# {{.history}}

# >>>>>>>>> Message/Query

# {{.query}}

# >>>>>>>>> Tool
# {{.tools}}

# >>>>>>>>> Result

# {{.result.Value | fromJson | toPrettyJson}}

# >>>>>>>>> Env

# {{printenv}}
# '

##
# script="file:///$PWD/swarm/resource/incubator/agents/meta/agent.yaml"
# script="file:///$PWD/swarm/resource/incubator/agents/web/agent.yaml"
# script="file:///$PWD/swarm/resource/incubator/agents/ralph/agent.yaml"

# message="write a report on the major world events for the year of 2025"
# message="What is the latest technology and research  human like consciousness for  AI and LLM"
# message="Stock market trend for the next one month"
# message="Plan an adventure to California for vacation"
# message="Plan a trip to China for a one month's vacation. My family would like to see famouse scenic sites."
# message="Plan a cruise trip for my family"
# message="suggestion for spending a day in Los Angels"
# message="what is the state of the art for LLM vibe coding and future developments in the next 2 years"
# message="what is the state of the art for LLM memory management and the future research in the next 2 years"
# message="Top open source github repo for comamnd line parsing in golang"

# /agent:web/search \
#     --option message="$message"
# agent="web/search"
# agent="web/react"

# tool="ralph:write_template_agent"
# tool="ralph:write_template_fix_plan"
# tool="ralph:write_template_prompt"

# /$tool  --script "$script" --base_dir=/tmp/test/xxx 

###

# export PATH=$PATH:/usr/local/go/bin:/usr/bin
# PATHx="/usr/bin:/go/bin:$PATH"
# /sh:set_envs --envs '{"HOME":"/Users/liqiang/", "PATH": "/usr/local/go/bin:/usr/bin", "GOPATH":"/Users/liqiang/go"}'
# /sh:set_envs --envs '{"HOME":"/Users/liqiang/","PATH": "/usr/local/go/bin:/usr/bin"}'

# printenv

# /sh:go --command "go version"
# /sh:go --command "go env"

# /sh:go --command "go build ./..."
# /bin/go
# echo ""

# /sh:exec --command "diff a b"
#
# function test_failed() {
#     echo "❌ test outputs differ"
#     exit 1
# }

/sh:exec --command "diff /tmp/test.txt ./test/data/test-expected.txt || echo  '❌❌ test outputs differ'" 
echo "status $?"

echo "***🎉🎉🎉  eval tests completed ***"
###