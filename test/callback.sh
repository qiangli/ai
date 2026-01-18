#!/usr/bin/env ai /sh:bash --format raw --script
set -xue


echo ">>> Testing sleep and callback..."

# sleep
sleep 3s

echo "$?"
echo "*** 🎉 Callback tests completed ***"
###