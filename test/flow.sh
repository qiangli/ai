#!/usr/bin/env ai /sh:bash --verbose --script
set -xue

echo "# Flow Test"
DRY=""

#
BASE=$(pwd)
echo $BASE
printenv

echo "## Sequence"

$DRY /flow:sequence --actions '["sh:parse", "ai:set_envs", "sh:format"]' --command='/ai:pass --option query="tell me a joke" --option error="no error" --option template="data:this is a test\n{{.result}} error: {{.error}}"' --verbose

# $DRY /flow:sequence --actions '["sh:parse", "fs:list_roots", "sh:format"]' --option command="/sh:pass" --option template="data:this\nis a test {{.result}} error: {{.error}}" --verbose

# $DRY /flow:sequence --actions '["sh:parse", "ai:call_llm", "sh:format"]' --option command="/ai:get_envs --option query='tell me a joke' --option template='data:this\nis a test {{.result}} error: {{.error}}'" --verbose

#
printenv
echo $?
exit 0
#
