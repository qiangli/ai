#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script

echo "Bash builtin system shell tests"

WS=$(/sh:workspace)
echo "*** workspace: $WS ***"

echo "*** pwd: $(pwd) $PWD ***"
ls /x || echo ">>>>>> /x does not exist"

env

date

# cd $workspace || echo "cd is not supported"
# For legacy bash scripts relying on `cd`, use the 'sh:exec' tool, e.g.,
/sh:exec --command "cd /tmp/ && ls -dl "
# # exec $PWD/test/bash-legacy.sh || echo "exec is not supported"
/sh:exec --command "bash $PWD/test/bash-legacy.sh"

ls -dl /tmp

echo "*** 🎉 Bash builtin system shell completed ***"
# ###