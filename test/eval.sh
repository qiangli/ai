#!/usr/bin/env ai /sh:bash --format raw --script
# set -ue

##

# // | fromJson | toPrettyJson
template='data:,
>>>>>>>>> Message
{{.query}}
>>>>>>>>> Instruction
{{.prompt}}

>>>>>>>>> Result
{{.result}}

>>>>>>>>> Env
{{printenv}}
'

##
# script="file:///$PWD/swarm/atm/resource/incubator/agents/meta/agent.yaml"
script="file:///$PWD/swarm/atm/resource/incubator/agents/gptr/agent.yaml"

message="write a report on the major world events for the year of 2025"

# preferences='
# {
#     "tone": "formal",
#     "total_words": "1000",
#     "report_format": "markdown",
#     "language": "english",
#     "original_query": "write a report on the major world events for the year of 2025"
# }
# '
# role_prompt='{"name":"cool_agent","display":"Cool Agent","instruction":"You are a smart agent!"}'
# scrape_result='["RESULT ONE","RESULT TWO"]'
# final_report="This is the report."

/flow:sequence \
    --agent "gptr/gptr" \
    --actions "[ai:spawn_agent,sh:format]" \
    --template "$template" \
    --script "$script" \
    --adapter "chat" \
    --message "$message"

    # --option echo__agent__gptr__user_input="$preferences" \
    # --option echo__agent__gptr__choose_agent="$role_prompt" \
    # --option echo__agent__search__scrape="$scrape_result" \
    # --option echo__agent__gptr__curate="$scrape_result" \
    # --option echo__agent__gptr__report="$final_report"


echo "---- script env ----"

printenv
# script="$PWD/swarm/atm/resource/incubator/agents/gptr/agent.yaml"

# # message="Plan an adventure to California for vacation"
# # message="Plan a trip to China for a one month's vacation. My family would like to see famouse scenic sites."
# # message="Plan a cruise trip for my family"
# message="suggestion for spending a day in Los Angels"

# # adapter="echo"
# # actions='[ai:new_agent,sh:format]'
# actions='[ai:spawn_agent,sh:format]'
# # actions='[sh:parse,sh:format]'

# # script="file:///$PWD/swarm/atm/resource/incubator/agents/search/agent.yaml"
# template='data:,
# >>> prompt
# {{.prompt|toPrettyJson}}

# >>> query
# {{.query|toPrettyJson}}

# >>> result
# {{.result}}

# >>> env
# {{printenv}}
# '

# preferences='
# {
#     "tone": "formal",
#     "total_words": "1000",
#     "report_format": "markdown",
#     "language": "english",
#     "original_query": "__original query__"
#     }
# '
# prompt='{"name":"cool_agent","display":"Cool Agent","instruction":"You are a smart agent!"}'
# result='["RESULT ONE","RESULT TWO"]'

# # agent="gptr/gptr"
# agent="gptr/plan_research"
# # agent="gptr/curate"
# # agent="gptr/report"

# # agent="search/scrape"
# /flow:sequence --agent "$agent" \
#     --actions "$actions"  \
#     --template "$template" \
#     --adapter "echo" \
#     --option query="$message" 

# --option preferences="$preferences" \
# --option agent_role_prompt="$prompt" \
# --option search_results="$result"

# "/flow:sequence",
# "--actions",
# "[ai:spawn_agent,sh:format]",
# "--agent",
# "gptr/plan_research",
# "--adapter",
# "echo",
# "--template",
# "data:,env: {{printenv}}",
# "--verbose",
# "--message",
# "tell me a joke"
# echo /flow:sequence --actions "[ai:spawn_agent,sh:format]" --agent "gptr/plan_research" --adapter "echo" --template "data:,env: {{printenv}}" --verbose --message "tell me a joke"

# echo "Publishing..."
# /flow:sequence --option actions="[agent:gptr/curate,agent:gptr/report]" --adapter "echo"


# /atm:hi --script "./swarm/atm/resource/template/atm.yaml" \
#     --info
# /agent:atm/hi --script "./swarm/atm/resource/template/atm.yaml" \
#     --adapter echo --info

# echo ""
echo "*** eval tests completed ***"
###