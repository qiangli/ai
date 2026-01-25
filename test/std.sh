#!/usr/bin/env ai /sh:bash --format raw --script
set -ue

# tools

tools=(
# ai
"ai:help"
"ai:call_llm"
"ai:list_agents"
"ai:get_agent_info"
"ai:read_agent_config"
"ai:new_agent"
# "ai:build_agent"
"ai:build_query"
"ai:build_prompt"
"ai:build_context"
"ai:transfer_agent"
# "ai:reload_agent"
"ai:spawn_agent"
"ai:list_tools"
"ai:get_tool_info"
"ai:read_tool_config"
"ai:list_models"
"ai:list_messages"
# "ai:get_message_info"
# "ai:save_messages"
"help:help"
"time:help"
"time:get_local_timezone"
# "time:convert_time"
# web
"web:ddg_search"
"web:bing_search"
"web:google_search"
"web:brave_search"
)

for tool in "${tools[@]}"; do
  /$tool --option adapter="echo" --output none \
      --option echo__${tool//:/__}="$tool ok"\
      --option query="what is up?" \
      --option agent="ed/ed" \
      --option tool="sh:bash" \
      --option echo__agent__ed__ed="ed ok"
done

# agents

agents=(
"help/help"
# context
"context/summary"
"context/lastn"
"context/summarizer"
#
"ask/ask"
"ed/ed"
# deep
"deep/deep"
"deep/memory"
# "deep/workspace"
# flow
"flow/flow"
"flow/sequence"
"flow/choice"
"flow/parallel"
"flow/shell"
# gptr
"gptr/gptr"
"gptr/preferences"
"gptr/curate"
"gptr/report"
# 
# "kbase/kbase"
# memory
"memory/memory"
"memory/long_term"
"memory/todo_list"
"memory/fold"
# meta
"meta/workspace"
# "meta/agent"
# "meta/prompt"
# "meta/dispatch"
# "meta/save_result"
# react
"react/react"
# research
"research/research"
"research/sub_agent"
"research/web_search"
"research/deep"
"research/critique"
# 
"shell/shell"
# swe
"swe/swe"
"swe/vibe"
# aider
"swe/architect"
"swe/ask"
"swe/code"
# gpte
"swe/generate"
"swe/improve"
"swe/clarify"
# 
# "think/think"
# web
"web/search"
"web/research"
"web/scrape"
)

for agent in "${agents[@]}"; do
  /agent:${agent} --option adapter="echo" --output none \
      --option echo__agent__${agent//\//__}="$agent ok" \
      --option query="what is up?"
done


#
# printenv

echo "$?"
echo "*** 🎉 standard/incubator tests completed ***"
exit 0
###

