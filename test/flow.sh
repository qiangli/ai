#!/usr/bin/env ai /sh:bash --format raw --script
# set -xue
set -ue


echo "# Flow Test"

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

adapter="echo"
# adapter="chat"

# command='/ai:pass --option query="tell me a joke" --option error="no error" --option n1=v1 --option n2=v2'
# /flow:sequence --actions '["sh:parse", "sh:format"]' --command="$command"
# /flow:sequence --actions '["sh:parse", "sh:set_envs", "sh:format"]' --command="$command" --template "$template"
# /flow:sequence --actions '["agent:ed"]' --option command="ls -al" --option adapter="$adapter" --option query="what is unix" --option template=$template

# # flow types
# /flow:sequence --actions '["/sh:pwd", "fs:list_roots", "agent:ed"]' --option command="ls -al" --option adapter="$adapter" --option query="what is unix" --option template=$template

# /flow:choice --actions '["/sh:pwd", "fs:list_roots", "agent:ed"]' --option command="ls -al" --option adapter="$adapter" --option query="what is unix" --option template=$template

# # /flow:parallel --actions '["/sh:pwd", "fs:list_roots", "agent:ed"]' --option command="ls -al" --option adapter="$adapter" --option query="what is unix" --option template=$template --option adapter="echo"
# /flow:parallel --actions '["/sh:format", "/sh:format", "/sh:format"]' --option query='query x' --template 'data:, *** {{.kit}}:{{.name}} query: {{.query}}' 

/flow:loop --actions "[sh:pwd,sh:pass]" --max-iteration 3 --sleep 1s --report "Runing in a loop"
# /flow:fallback --actions '["/sh:pwd", "fs:list_roots", "agent:ed"]' 

#
echo "$?"
echo "*** 🎉 flow tests completed ***"
exit 0
#
