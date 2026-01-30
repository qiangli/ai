#!/usr/bin/env bash

echo ">>> Classic Bash tests - no extended feature support for agent/tool"

set -eu

# cd is supported
cd /tmp
pwd

# tool not supported
/fs:list_roots || echo "*** Agent/tool not supported"

echo ">>> Classic Bash tests completed ***"
# ###