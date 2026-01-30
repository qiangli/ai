#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script

# echo "Bash builtin system shell tests"

# set -o pipefail

# # cd $workspace || echo "cd is not supported"
# ls /x || echo ">>>>>> /x does not exist"

# pwd
# For legacy bash scripts relying on `cd`, use the 'sh:exec' tool, e.g.,
# /sh:exec --command "/bin/bash $PWD/test/bash-legacy.sh"


# WS=$(/sh:workspace)
/sh:workspace
# echo "*** workspace:  ***"
# /fs:list_roots
# printenv

# echo "*** 🎉 Bash builtin system shell completed ***"
# ###