#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script

# test template vars: env, args, and params
set -ue

# // | fromJson | toPrettyJson
template='data:,
>>>>>>>>> Instruction/Prompt
---
{{.prompt}} <-- this is cleared after call_llm
---
>>>>>>>>> Context/History
---
%%%%%%%%%%%%%%%%%%%%%%
{{ range .history }} <-- this is cleared after call_llm
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
# script="file:/$PWD/test/agent/agent.yaml"
script="file:./test/agent/agent.yaml"

message="tell me a joke"
echo "script: $script"
# env message="${message}"
# env datetime="<TODO>"
# # env workspace="<redacted>"
# env input="<redated>"
# # env HOME="<redated>"
# env PATH="/bin:/usr/bin"

# /fs:list_roots --resolve
# TODO only support /sh:set_envs, ignore: os.Setenv()
# /sh:get_envs

/flow:sequence \
    --actions "[ai:spawn_agent,sh:format]" \
    --agent "test/test" \
    --adapter "echo" \
    --template "$template" \
    --script "$script" \
    --output "file:/tmp/test.txt"

###

printenv
# echo ""
echo "*** 🎉 test [envs, args, params] completed ***"
###