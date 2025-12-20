#!/usr/bin/env ai /sh:bash --format raw --script
# set -xue
set -ue

# echo "### sb.sh"
# echo "$$ I'm called $@..."
# echo "Test inline data url"
# echo "tool action, custom bash command, bash builtin, bash command, local command"
# /sh:bash --script 'data:,#!\nset -x\n/fs:list_roots\nprintenv\necho this is a test\nls -al\ngo version'
# /fs:list_roots --arg query="hello"

echo "sleeping 10 sec..."
/sh:bash --script 'data:,#!\nset -x\n/sleep 10'

# /sh:timeout --action "/sh:bash" --arg duration="3s" --script 'data:,#!\nset -x\n/sleep 10'
