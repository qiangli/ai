#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script

echo "Bash builtin system shell tests"

set -ue

cd ./
pwd

# name=value
# export name
# export VAR=VAL
##
# array="[HOME=~,PATH=/usr/bin]"
# /sh:set_envs --option envs="${array}"
/sh:get_envs
# /sh:list_roots

echo "*** 🎉 Bash builtin system shell completed ***"
# ###