#!/usr/bin/env ai
set -xue

echo "# Hello world!"
DRY="echo"

echo "## system command..."
ID="ID:$$"
echo $ID
BASE=$(pwd)
echo $BASE
printenv
/bin/pwd
/bin/ls -al /opt

echo "## tool/bash..."

/sh:exec --command "ls -al /opt"
/sh:bash --command "ls -al /opt"

# TODO support subshell or print file content
# /sh:bash --script "data:$(cat $BASE/test/sub.sh)"
/sh:bash --script "$BASE/test/sub.sh"
/ai:execute_tool --tool "sh:exec" --command "ls -al /opt"
/ai:execute_tool --tool "sh:bash" --command "ls -al /opt"

#
echo "## agent..."
$DRY /agent:ed "Agent \"ed-${ID}\": correcto mine englise please."
$DRY /ai:spawn_agent --agent "joker"  --message "what is on the news today"

#
echo "## tool/agent from custom content..."

atm_script="$(pwd)/swarm/atm/resource/template/atm.yaml"
echo $atm_script

/atm:hi --script $atm_script --arg greeting="how are you today?" --arg names='["dragon", "horse"]'
$DRY /agent:atm/hi --script $atm_script

#
echo $?
exit 0
#
