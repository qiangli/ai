#!/usr/bin/env ai /sh:bash --format raw --script
# set -ue

# // | fromJson | toPrettyJson
template='data:,
>>>>>>>>> Instruction/Prompt
---
{{.prompt}}
---
>>>>>>>>> Context/History
---
{{.history}}
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

script="file:///$PWD/swarm/atm/resource/incubator/agents/test/agent.yaml"
message="tell me a joke"

env message="${message}"
env datetime="<TODO>"

# # /agent:test/test \
# #     --script "$script" \
# #     --adapter "echo" \
# #     --message "$message"

/flow:sequence \
    --agent "test/test" \
    --actions "[ai:spawn_agent,sh:format]" \
    --adapter "echo" \
    --template "$template" \
    --script "$script" \
    --option output="file:/tmp/test.txt"

###

printenv
# /sh:get_envs --option keys='["result"]'

# printf "result: %s\n" "$message"

# date
# echo ""
echo "*** test completed ***"
###