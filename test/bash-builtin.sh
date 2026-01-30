#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script

# echo "Bash builtin system shell tests"

# cd $workspace || echo "cd is not supported"
# For legacy bash scripts relying on `cd`, use the 'sh:exec' tool, e.g.,
exec $PWD/test/bash-legacy.sh
# /sh:exec --command "/bin/bash $PWD/test/bash-legacy.sh"


# WS=$(/sh:workspace)
# echo "*** pwd: $(pwd) workspace: $WS ***"
# # /fs:list_roots
# # printenv
# ls /x || echo ">>>>>> /x does not exist"

# echo "*** 🎉 Bash builtin system shell completed ***"
# ###