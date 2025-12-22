#!/usr/bin/env ai /sh:bash --format raw --script
set -xue

# echo "### sb.sh"
# echo "$$ I'm called $@..."
# echo "Test inline data url"
# echo "tool action, custom bash command, bash builtin, bash command, local command"
# /sh:bash --script 'data:,#!\nset -x\n/fs:list_roots\nprintenv\necho this is a test\nls -al\ngo version'
# /fs:list_roots --option query="hello"

echo "testing..."
# time /bin/ls -al

# time tests
# time /sh:exec --command "sleep 5"
# time /sh:bash --script "data:,sleep 3"
# time /agent:joker --message "explain timeout in a unix system" --adapter "echo"

# timeout tests
/sh:timeout --command "/sh:exec --command 'sleep 10'"  --option duration="3s"
# /sh:timeout --command "/sh:bash --script 'data:,sleep 10'"  --option duration="3s"
# /sh:timeout --command "/agent:joker --message 'explain timeout in a unix system'" --option duration="3s"

