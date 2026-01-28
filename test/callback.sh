#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script

echo ">>> Testing sleep and callback..."

set -euo pipefail

# sleep
# sleep 3s

printenv

# if [[ -z "${action:-}" ]]; then
#     echo "ai:callback: missing required parameter: action" #>&2
#     exit 2
# fi

# if [[ "${action}" != /* ]]; then
#     echo "ai:callback: invalid action: must begin with '/' (slash command syntax), got: ${action}" >&2
#     exit 3
# fi

# if [[ -n "${wait:-}" ]]; then
#     /ai:sleep --duration "${wait}"
# fi

# # Execute the action as a single command string to preserve spacing/quoting.
# eval "${action}"
# exit $?
exit 0


# echo "$?"
# echo "*** 🎉 Callback tests completed ***"
###