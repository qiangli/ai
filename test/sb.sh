#!/usr/bin/env ai --base ./test/data/ 
# set -x
echo "### sb.sh"
echo "$$ I'm called $@..."
echo "Test inline data url"
echo "tool action, custom bash command, bash builtin, bash command, local command"
/sh:bash --script 'data:,#!\nset -x\n/fs:list_roots\nprintenv\necho this is a test\nls -al\ngo version'
/fs:list_roots --option query="hello"
echo "return..."
