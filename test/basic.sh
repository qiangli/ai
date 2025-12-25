#!/usr/bin/env ai /sh:bash --format raw --script

echo "bash support compatibility tests"

set -xueo pipefail

echo ">>> Testing basic..."

echo "script path":
echo $0
# TODO
# echo "$BASH_SOURCE"

echo "script name: $(dirname $0) $(basename $0)"

ls -al /tmp; echo "status: $?"

##
ls /tmp && echo "/tmp exists"
if ls /tmp; then
    echo "/tmp exists"
fi

# TODO
# ls /xyz || echo "/xyz does not exist"
# if ! ls /xyz; then
#     echo "/xyz does not exist"
# fi

if [[ $? -eq 0 ]]; then
  echo "double bracket OK"
fi

if [ $? -eq 0 ]; then
  echo "single bracket OK"
fi

if test $? -eq 0; then
  echo "test OK"
fi

##
case $? in
    0) echo "OK";;
    1) echo "Error";;
    *) echo "Unhandled error";;
esac

function foobar() {
    echo "foo $1"
    echo "bar $2"
}

foobar "first" "second"

function loop() {
    for arg in "$@"; do
        echo "arg: $arg"
    done
}

loop 1 2 3 4

function at() {
    printf '@: [%s]\n' "$@"
    echo at n: ${#@}
}

function star() {
    printf '*: [%s]\n' "$*"
    echo star n: ${#*}
}

at "1 2" "3 4 5"
star "1 2" "3 4 5" 

echo uid: $UID euid: $EUID 

env

# // bash commands
# 	"true", "false", "exit", "set", "shift", "unset",
# 	"echo", "printf", "break", "continue", "pwd", "cd",
# 	"wait", "builtin", "trap", "type", "source", ".", "command",
# 	"dirs", "pushd", "popd", "umask", "alias", "unalias",
# 	"fg", "bg", "getopts", "eval", "test", "[", "exec",
# 	"return", "read", "mapfile", "readarray", "shopt",
# 

$(exec "/bin/ls -al")

# // internal commands
# 	"base64", "basename", "cat", "chmod", "cp", "date", "dirname", "find", "gzip", "head", "ls", "mkdir",
# 	"mktemp", "mv", "rm", "shasum", "sleep", "tac", "tail", "tar", "time", "touch", "wget", "xargs",
#
base64 go.mod
basename $PWD 
cat go.mod
chmod a+x test/*.sh
cp test/script.sh /tmp/
date
dirname $PWD 
find ./test -name script.sh
# # gzip
head -n 3 test/script.sh
ls -ahdlQRFS ./test
mkdir -p /tmp/test
mktemp mk-temp-test
touch /tmp/test/hi
mv /tmp/test/hi /tmp/test/hello
rm /tmp/test/hello
shasum go.mod
sleep 1s
tac go.mod
tail -n 3 go.mod
# # tar
time touch /tmp/test/bye
wget -O /tmp/test/out.txt https://www.google.com
# xargs

echo "*** Basic tests completed ***"
###
