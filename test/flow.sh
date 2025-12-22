#!/usr/bin/env ai /sh:bash --format raw --script
# set -xue
set -ue


echo "# Flow Test"
DRY=""

#
BASE=$(pwd)
echo $BASE
printenv

echo ""
echo ">>> Sequence Test <<<"
echo ""

# {{.result.Value |fromJson |toPrettyJson}}
template='data:,
*** Query:
{{.query}}

*** Result:
{{.result}}

*** Error:
{{.error}}

*** Environment:
{{printenv}}
'

# command='/ai:pass --option query="tell me a joke" --option error="no error" --option n1=v1 --option n2=v2'
# $DRY /flow:sequence --actions '["sh:parse", "sh:format"]' --command="$command"
# $DRY /flow:sequence --actions '["sh:parse", "sh:set_envs", "sh:format"]' --command="$command" --template "$template"

# flow types
# $DRY /flow:sequence --actions '["/sh:pwd", "fs:list_roots", "agent:ed"]' --option command="ls -al" --option adapter="echo" --option query="what is unix" --option template=$template

# $DRY /flow:choice --actions '["/sh:pwd", "fs:list_roots", "agent:ed"]' --option command="ls -al" --option adapter="echo" --option query="what is unix" --option template=$template

# $DRY /flow:parallel --actions '["/sh:pwd", "fs:list_roots", "agent:ed"]' --option command="ls -al" --option adapter="echo" --option query="what is unix" --option template=$template --option adapter="echo"
# $DRY /flow:parallel --actions '["/sh:format", "/sh:format", "/sh:format"]' --option query='query x' --template 'data:, *** {{.kit}}:{{.name}} query: {{.query}}' 

# $DRY /flow:map --actions '["sh:format"]' --option query='["a", "b", "c"]' --template 'data:, *** {{.kit}}:{{.name}} input: {{.query}}' 

# $DRY /flow:chain --actions '["sh:pwd", "fs:list_roots", "agent:ed", "sh:format"]'  --option adapter="echo" --option query="what is unix" --option template=$template
$DRY /flow:chain --actions '["sh:pwd", "fs:list_roots", "agent:ed", "sh:format"]'  --option adapter="echo" --option query="what is unix" --option template='data:,{{.query}}'

# $DRY /flow:sequence --actions '["sh:parse", "ai:call_llm", "sh:format"]' --option command="/ai:get_envs --option query='tell me a joke' --option template='data:this\nis a test {{.result}} error: {{.error}}'" --verbose

#
# printenv
# echo $?
exit 0
#
