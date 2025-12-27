#!/usr/bin/env ai /sh:bash --format raw --script

echo "Bash basic extension tests"

set -xue

##
/bin/ls -al /tmp

echo ">>> Testing bash extnesions..."

# system command
/sh:exec --command "ls -al /tmp"

# tool
/atm:hi --script "$(pwd)/swarm/atm/resource/template/atm.yaml" \
     --option greeting="how are you today?" \
     --option names='["dragon", "horse"]'

# agent
# /agent:atm/hi --script "$(pwd)/swarm/atm/resource/template/atm.yaml" \
#     --adapter echo

# alias
/alias:cmd --option cmd="ls -al /tmp"

# scripting
/sh:bash --script "data:,ls -al /tmp"
/sh:bash --script "test/sb.sh"
# TODO support subshell or print file content
# /sh:bash --script "data:,$(cat $BASE/test/sb.sh)"

# 
# /ai:execute_tool --tool "sh:exec" --command "ls -al /opt"
# /ai:execute_tool --tool "/alias:agent_ed" \
#     --option agent_ed='/agent:ed --adapter echo --message "correcto mine englise."'

# /ai:spawn_agent --adapter echo --agent ed --message "correcto yours esperanto"

# flow
/flow:sequence --actions '["sh:pass","sh:pwd"]'
/flow:choice --actions '[sh:pwd,sh:pass]'
/flow:parallel --actions '[sh:pwd,sh:pass]'
/flow:map --actions '["sh:format"]' --option query='["a", "b", "c"]' --template 'data:, *** {{.kit}}:{{.name}} input: {{.query}}' 
/flow:chain --option chain=[sh:timeout,sh:backoff,alias:cmd] \
    --option cmd="ls -al /tmp" \
    --option duration="60s"

echo "*** Bash extention tests completed ***"
###