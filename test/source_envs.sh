#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script
set -ue

BASE=$(pwd)
echo "Running source_envs test from $BASE"

# Path to the env file (created in test/data)
ENV_FILE="$BASE/test/data/source_envs.env"

echo "Using env file: ${ENV_FILE}"

# Invoke the new tool to source environment variables from the file
/sh:source_envs --source "file:${ENV_FILE}"

echo "--- after /sh:source_envs ---"
# Print the variables that should have been set (or empty if not set)
echo "FOO=${FOO-}"
echo "BAZ=${BAZ-}"
echo "GOOD=${GOOD-}"
echo "PATH=${PATH-}"
echo "EMPTY=${EMPTY-}"
echo "SPACED=${SPACED-}"
echo "QUOTED=${QUOTED-}"
echo "WITH_EQUALS=${WITH_EQUALS-}"
# printenv

echo "*** source_envs test completed ***"
