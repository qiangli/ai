#!/usr/bin/env ai
# set -x
echo "### sub.sh"
echo "$$ I'm called $@..."
echo "Test inline data url"
echo "tool action, custom bash command, bash builtin, bash command, local command"
/sh:bash --script 'data:,#!\nset -x\n/fs:list_roots\nprintenv\necho this is a test\nls -al\ngo version'
/fs:list_roots --arg query="hello"
echo "return..."
