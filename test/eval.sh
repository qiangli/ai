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
# script="file:///$PWD/swarm/atm/resource/incubator/agents/meta/agent.yaml"
script="file:///$PWD/swarm/atm/resource/incubator/agents/web/agent.yaml"

# message="write a report on the major world events for the year of 2025"
# message="What is the latest technology and research  human like consciousness for  AI and LLM"
# message="Stock market trend for the next one month"
# message="Plan an adventure to California for vacation"
# message="Plan a trip to China for a one month's vacation. My family would like to see famouse scenic sites."
# message="Plan a cruise trip for my family"
# message="suggestion for spending a day in Los Angels"
# message="what is the state of the art for LLM vibe coding and future developments in the next 2 years"
# message="what is the state of the art for LLM memory management and the future research in the next 2 years"
message="Top open source github repo for comamnd line parsing in golang"

# /agent:web/search \
#     --option message="$message"

/flow:sequence \
    --agent "ask/ask" \
    --actions "[ai:spawn_agent,sh:format]" \
    --adapter "echo" \
    --template "$template" \
    --script "$script" \
    --message "$message"

###

printenv

# echo ""
echo "*** eval tests completed ***"
###