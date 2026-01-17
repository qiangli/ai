#!/usr/bin/env ai /sh:bash --format raw --script

# Bash Test

set -xueo pipefail

## Standard Syntax

### conditional
if [[ $? -eq 0 ]]; then
  echo "double bracket OK"
fi
if [ $? -eq 0 ]; then
  echo "single bracket OK"
fi
if test $? -eq 0; then
  echo "test OK"
fi

### switch
case $? in
    0) echo "OK";;
    1) echo "Error";;
    *) echo "Unhandled error";;
esac

### function
function foobar() {
    echo "foo $1"
    echo "bar $2"
}
foobar "first" "second"

### loop
function loop() {
    for arg in "$@"; do
        echo "arg: $arg"
    done
}
loop 1 2 3 4

### exec
$(exec "/bin/ls -al")

### system commands (limited support)
# "base64", "basename", "cat", "date", "dirname", "head", "ls",
# "shasum", "sleep", "tail", "time", "touch",
#
FILE="/tmp/test.txt"
touch $FILE
#
base64 $FILE
basename $PWD 
cat $FILE
date
dirname $PWD 
head -n 3 $FILE
ls -ahdlQRFS $PWD
shasum $FILE
sleep 1s
tail -n 3 $FILE
time touch $FILE

## Bash extension - run agents and tools

## tool - /KIT:NAME
/ai:help

## agent - /agent:PACK/NAME
# use "chat" or other adapters for real LLM  call.
# default is "chat" if not specified
# "echo" adapter is for testing only which echoes the request.
/agent:root/root --adapter "echo" --query "Hello"

## system command
/sh:exec --command "ls -al /tmp"

## alias - "cmd"
/alias:cmd --option cmd="ls -al /tmp"

# script - inline data: or file:
/sh:bash --script "data:,ls -al /tmp"
# /sh:bash --script "<path>"

# flow control tool
/flow:sequence --actions '["sh:pass","sh:pwd"]'
/flow:choice --actions '[sh:pwd,sh:pass]'
/flow:parallel --actions '[sh:pwd,sh:pass]'
/flow:chain --option chain=[sh:timeout,sh:backoff,alias:cmd] \
    --option cmd="ls -al /tmp" \
    --option duration="60s"


echo "*** 🎉 Example tests completed ***"
exit 0
###
