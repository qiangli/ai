#!/usr/bin/env ai /sh:bash --verbose --script
set -xue

echo "# Flow Test"
DRY=""

#
BASE=$(pwd)
echo $BASE
printenv

echo "## Sequence"

$DRY /sh:flow --actions '["sh:parse", "ai:get_envs", "sh:format"]' --arg command="/fs:list_roots" --arg template="data:this\nis a test {{.result}} error: {{.error}}" --verbose

$DRY /sh:flow --actions '["sh:parse", "ai:call_llm", "sh:format"]' --arg command="/ai:get_envs --arg query='tell me a joke' --arg template='data:this\nis a test {{.result}} error: {{.error}}'" --verbose

#
printenv
echo $?
exit 0
#
