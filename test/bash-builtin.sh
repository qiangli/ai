#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script

echo "Bash builtin system shell tests"

# set -ue
# oldpwd=$(pwd)
cd $workspace
pwd

# # printenv

# cd $oldpwd
# # name=value
# # export name
# # export VAR=VAL
# ##
# # array="[HOME=~,PATH=/usr/bin]"
# # /sh:set_envs --option envs="${array}"
/sh:get_envs
/fs:list_roots
printenv

echo "*** 🎉 Bash builtin system shell completed ***"
# ###