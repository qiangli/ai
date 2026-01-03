#!/usr/bin/env ai /sh:bash --format raw --script
# set -ue

# // | fromJson | toPrettyJson
template='data:,
>>>>>>>>> Message

{{.query}}

>>>>>>>>> Instruction

{{.prompt}}

>>>>>>>>> Result


>>>>>>>>> Env

{{printenv}}
'

##
# script="file:///$PWD/swarm/atm/resource/incubator/agents/meta/agent.yaml"
# script="file:///$PWD/swarm/atm/resource/incubator/agents/gptr/agent.yaml"
# script="$PWD/swarm/atm/resource/incubator/agents/gptr/agent.yaml"
# script="file:///$PWD/swarm/atm/resource/incubator/agents/search/agent.yaml"
# script="file:///$PWD/swarm/atm/resource/incubator/agents/research/agent.yaml"
# script="file:///$PWD/swarm/atm/resource/incubator/agents/deep/agent.yaml"

script="file:///$PWD/swarm/atm/resource/incubator/agents/test/agent.yaml"

# message="write a report on the major world events for the year of 2025"
# message="What is the latest technology and research  human like consciousness for  AI and LLM"
# message="Stock market trend for the next one month"
# message="Plan an adventure to California for vacation"
# message="Plan a trip to China for a one month's vacation. My family would like to see famouse scenic sites."
# message="Plan a cruise trip for my family"
# message="suggestion for spending a day in Los Angels"
# message="what is the state of the art for LLM vibe coding and future developments in the next 2 years"
message="what is the state of the art for LLM memory management and the future research in the next 2 years"

# /flow:sequence \
#     --agent "deep/deep" \
#     --actions "[ai:spawn_agent,sh:format]" \
#     --adapter "echo" \
#     --template "$template" \
#     --script "$script" \
#     --message "$message"

# /agent:deep \
#     --script "$script" \
#     --option  query="$message"

# /agent:research \
#     --script "$script" \
#     --option  query="$message"

# /agent:research/sub_agent \
#     --script "$script" \
#     --option  query="$message"

# /agent:research/web_search \
#     --script "$script" \
#     --option  query="$message"

# /research:think_tool \
#     --script "$script" \
#     --option  reflection="$message"

/agent:test/test \
    --script "$script" \
    --option  query="$message"

###
# echo "---- script env ----"

# printenv

# echo ""
# echo "*** eval tests completed ***"
###