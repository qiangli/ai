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

# /sh:set_envs --option query="$message"
# fix me
# datetime=$(date)
datetime="Jan  1 08:08:08"
/sh:set_envs --option datetime="$datetime"

# /agent:test/test \
#     --script "$script" \
#     --adapter "echo" \
#     --message "$message"

# /flow:sequence \
#     --agent "test/test" \
#     --actions "[ai:spawn_agent,sh:format]" \
#     --adapter "echo" \
#     --template "$template" \
#     --script "$script" \
#     --message "$message"

###

# printenv
date
echo ""
echo "*** test completed ***"
###