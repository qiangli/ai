#!/usr/bin/env ai /sh:bash --format raw --script

# test template vars: env, args, and params
set -ue

# // | fromJson | toPrettyJson
template='data:,
>>>>>>>>> Instruction/Prompt
---
{{.prompt}}
---
>>>>>>>>> Context/History
---
%%%%%%%%%%%%%%%%%%%%%%
{{ range .history }}
Content: empty? {{ empty .Content }}
Role: {{ .Role }}
{{ end }}
%%%%%%%%%%%%%%%%%%%%%%
---
>>>>>>>>> Message/Query
---
{{.query}}
---
>>>>>>>>> Result
---
---
>>>>>>>>> Env
---
{{printenv}}
---
'

##
script="file:///$PWD/swarm/resource/incubator/agents/test/agent.yaml"
message="tell me a joke"

env message="${message}"
env datetime="<TODO>"
env workspace="<redacted>"

/flow:sequence \
    --agent "test/test" \
    --actions "[ai:spawn_agent,sh:format]" \
    --adapter "echo" \
    --template "$template" \
    --script "$script" \
    --option output="file:/tmp/test.txt"

###

printenv
# echo ""
echo "*** 🎉 test [envs, args, params] completed ***"
###