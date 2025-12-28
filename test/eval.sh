#!/usr/bin/env ai /sh:bash --format raw --script
set -ue

# adapter="chat"

# {{.result.Value |fromJson |toPrettyJson}}
# actions='["ai:read_agent_config","ai:new_agent","ai:build_context","ai:call_llm","sh:format"]'
# template='data:,
# >>> Query:
# {{.query}}

# >>> Result:
# {{.result |toPrettyJson}}

# >>> Error:
# {{.error}}

# ---
# {{.agent}}
# ---
# {{printenv}}
# '


# ##
# actions='[ai:spawn_agent,sh:format]'
# agent="memory/memory"
# # agent="memory/long_term"
# # agent="memory/todo_list"
# template='data:,
# >>>>>>>>>
# {{.prompt}}

# >>>>>>>>>
# {{printenv}}
# '

# ##
# # adapter="chat"
# actions='[ai:spawn_agent,sh:format]'
# agent="context/lastn"
# # agent="context/summary"
# template='data:,
# >>>>>>>>>
# {{.history | toPrettyJson}}

# >>>>>>>>>
# {{printenv}}
# '

# ##
# actions='[ai:spawn_agent,sh:format]'
# agent="kbase"
# template='data:,
# >>>>>>>>>
# {{.prompt}}

# {{.tools | toPrettyJson}}
# >>>>>>>>>
# {{printenv}}
# '


# ##
# actions='[ai:spawn_agent,sh:format]'
# agent="think"
# template='data:,
# >>>>>>>>>
# {{.prompt}}

# >>>>>>>>>
# {{.tools | toPrettyJson}}

# >>>>>>>>>
# {{.model | toPrettyJson}}

# >>>>>>>>>
# {{printenv}}

# '

##

# actions='[ai:spawn_agent,sh:format]'

# template='data:,

# >>>>>>>>> Instruction/Prompt
# {{.prompt}}

# >>>>>>>>> Context/History
# {{.history }}

# >>>>>>>>> Message/Query
# {{.query}}

# >>>>>>>>> Tools
# {{.tools }}

# >>>>>>>>> Model
# {{.model }}

# >>>>>>>>> Environment
# {{printenv}}

# '

# agent="aider/architect"
# extra="--message write a simple hello world"

# agent="gptr/user_input"
# agent="gptr/choose_agent"
# agent="gptr/save_response"
# agent="gptr/web_search"
# agent="gptr/research_queries"
# agent="gptr/crawl"
# agent="gptr/scrape"
# agent="gptr/curate"
# agent="gptr/report"

# extra="--message write a report on the latest LLM updates"

# /flow:sequence --actions "$actions" --template "$template" --agent "$agent" --adapter "$adapter" $extra

# ##
# actions='[ai:read_agent_config,gptr:research,sh:format]'
# script="file:///$PWD/swarm/atm/resource/incubator/agents/gptr/agent.yaml"
# template='data:,
# >>>>>>>>> 
# '
# /flow:sequence --actions "$actions" --template "$template" --script "$script" --adapter "$adapter" $extra

# agent="gptr/gptr"
# export message="write a report on the major world events for the year of 2025"
# /agent:gptr/gptr -adapter "chat" --message "$message"

# template='data:,
# {{.original_query}}
# '

# /sh:set_envs --option original_query="$message"
# /sh:get_envs --option keys="[original_query]" --template "$template"
# # 
# echo "Creating task specific agent and setting up default values..."
# /flow:parallel --option actions="[agent:gptr/user_input,agent:gptr/choose_agent]" --adapter "chat" --message "$message"
# echo ">>> environemtn"

# script="$PWD/swarm/atm/resource/incubator/agents/meta/agent.yaml"
# /agent:meta/prompt --adapter "chat" --script "$script" --option query="$message" --option n-queries=18
# /agent:meta/dispatch --adapter "echo" --script "$script" --option query="$message" --option n-queries=18
# /agent:meta/agent --adapter "chat" --script "$script" --option query="$message" --option n-queries=18

script="$PWD/swarm/atm/resource/incubator/agents/search/agent.yaml"
# message="Plan an adventure to California for vacation"
# message="Plan a trip to China for a one month's vacation. My family would like to see famouse scenic sites."
message="Plan a cruise trip for my family"

# /agent:search/scrape --adapter "echo" --script "$script" --option query="$message" --option n-queries=18


# printenv
# /sh:get_envs --option keys="[prompt]" --format "data:,{{.prompt|toPrettyJson}}"

# echo "Researching..."
adapter="chat"
# actions='[ai:new_agent,ai:build_query,sh:format]'
actions='[ai:spawn_agent,sh:format]'
# script="file:///$PWD/swarm/atm/resource/incubator/agents/search/agent.yaml"
template='data:,
>>> prompt
{{.prompt|toPrettyJson}}

>>> query
{{.query|toPrettyJson}}

>>> result
{{.result}}

>>> env
{{printenv}}
'
extra="--message $message"
/flow:sequence --agent "search/scrape" --actions "$actions" --template "$template" --script "$script" --adapter "$adapter" $extra

# echo "Publishing..."
# /flow:sequence --option actions="[agent:gptr/curate,agent:gptr/report]" --adapter "echo"


# /atm:hi --script "./swarm/atm/resource/template/atm.yaml" \
#     --info
# /agent:atm/hi --script "./swarm/atm/resource/template/atm.yaml" \
#     --adapter echo --info

echo ""
echo "*** eval tests completed ***"
###