#!/usr/bin/env ai /sh:bash --format raw --script
set -xue

echo "# Hello world!"

echo "## system command..."
ID="ID:$$"
echo $ID
BASE=$(pwd)
echo $BASE
printenv
/bin/pwd
#
/bin/ls -al /opt
export cmd="ls -al /opt"
/sh:bash --option script="data:,$cmd"
# #TODO expandenv for scripts: /sh:bash --option script='data:,$cmd'

echo "## tool/bash..."

/sh:exec --command "ls -al /opt"
/sh:bash --script "data:,ls -al /opt"

# TODO support subshell or print file content
# /sh:bash --script "data:,$(cat $BASE/test/sb.sh)"
/sh:bash --script "$BASE/test/sb.sh"
/ai:execute_tool --tool "sh:exec" --command "ls -al /opt"
/ai:execute_tool --tool "sh:bash" --command "ls -al /opt"

#
adapter="--adapter echo"
echo "## agent..."
/agent:ed ${adapter} "Agent \"ed-${ID}\": correcto mine englise please."
/ai:spawn_agent ${adapter} --agent joker --message "what is on the news today"

#
echo "## tool/agent from custom content..."

atm_script="$(pwd)/swarm/atm/resource/template/atm.yaml"
echo $atm_script

/atm:hi --script $atm_script --option greeting="how are you today?" --option names='["dragon", "horse"]'
echo /agent:atm/hi ${adapter} --script $atm_script

#
echo "$?"
echo  "*** script tests completed ***"
exit 0
#
