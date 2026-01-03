#!/usr/bin/env ai /sh:bash --format raw --script
# set -ue

# // | fromJson | toPrettyJson
template='data:,
>>>>>>>>> Instruction/Prompt

{{.prompt}}

>>>>>>>>> Context/History

{{.history}}

>>>>>>>>> Message/Query

{{.query}}

>>>>>>>>> Result

>>>>>>>>> Env

{{printenv}}
'

##

script="file:///$PWD/swarm/atm/resource/incubator/agents/test/agent.yaml"
message="what is the state of the art for LLM memory management and the future research in the next 2 years"

# /agent:test/test \
#     --script "$script" \
#     --adapter "echo" \
#     --message "$message"
# /sh:set_envs --option query="$message"
/flow:sequence \
    --agent "test/test" \
    --actions "[ai:spawn_agent,sh:format]" \
    --adapter "echo" \
    --template "$template" \
    --script "$script"

###

printenv

echo ""
echo "*** test completed ***"
###