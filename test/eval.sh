#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script

set -xu

# BASE=$(pwd)
# /bin/ls /x
# ls /x
# cd /
# exec test.sh
# /sh:fail
# /fs:list_roots
./test/env.sh
/bin/bash ./test/bash-legacy.sh
echo "status $?"

# echo "continued"
###