#!/usr/bin/env ai /sh:bash --verbose --format raw --script
set -xue

# actions='["ai:read_tool_config","sh:format"]'
# /sh:flow --actions "$actions" --template "data:,{{toPrettyJson .config}}" --tool "fs:read_file" --format raw

actions='["ai:read_agent_config","sh:format"]'
/sh:flow --actions "$actions" --template "data:,{{toPrettyJson .config}}" --agent "ed" --format raw

# /ai:call_llm --message "what is the headlines today in the news" --instruction "you are a top news anchor. if need to access the web and other source, use the ai:list_tools to find the right tools to use" --arg tools='["ai:list_tools","ai:execute_tool","web:get_web_content","web:ddg_search"]'
