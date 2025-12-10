#!/usr/bin/env ai
set -x

echo "Hello world!"

printenv

# 
/bin/ls

echo $?

/bin/pwd

echo $?

/sh:bash --command "ls -al"

echo $?

/sh:bash --command "pwd"

echo $?

/agent:ed "correcto mine englise"

echo $?

exit 0
#
