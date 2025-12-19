#!/usr/bin/env ai /sh:bash --format raw --script
# set -xue
set -ue

adapter="echo"

# actions='["ai:read_tool_config","sh:format"]'
# /sh:flow --actions "$actions" --template "data:,{{toPrettyJson .config}}" --tool "fs:read_file" --format raw

# actions='["ai:read_agent_config","sh:format"]'
# /sh:flow --actions "$actions" --template "data:,{{toPrettyJson .config}}\n\n agent: {{.agent}}\n kit: {{.kit}}\ nname: {{.name}}\n" --agent "ed" --format raw

# actions='["ai:read_agent_config", "ai:new_agent", "sh:format"]'
# /sh:flow --actions "$actions" --template "data:,{{toPrettyJson .agent}} \n kit: {{.kit}} \n name: {{.name}}" --agent "ed" --format raw

# actions='["ai:read_agent_config", "ai:new_agent", "ai:call_llm", "sh:format"]'
# /sh:flow --actions "$actions" --template "data:,###\n****************{{toPrettyJson .result}}\n*************" --agent "ed" --message "correcto mine englise" --adapter "$adapter" --format raw

# actions='["ai:read_agent_config", "ai:new_agent", "ai:build_query", "ai:call_llm", "sh:format"]'
# /sh:flow --actions "$actions" --template "data:,###\n****************{{.query}}\n*************" --agent "ed" --message "correcto mine englise{{date}} agent: {{.agent}} adapter{{.adapter}}" --adapter "$adapter"

actions='["ai:read_agent_config"]' #, "ai:new_agent"]' #, "ai:build_query"]' #, "sh:format"]'
/sh:flow --actions "$actions" --template "data:,###\nhello" --agent "ed" --message "correcto mine englise"  #--instruction "you are fantastic  agent: adapter" --adapter "$adapter"

# /ai:call_llm --message "what is the headlines today in the news" --instruction "you are a top news anchor. if need to access the web and other source, use the ai:list_tools to find the right tools to use" --arg tools='["ai:list_tools","ai:execute_tool","web:get_web_content","web:ddg_search"]'
