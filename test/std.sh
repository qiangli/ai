#!/usr/bin/env ai /sh:bash --format raw --script
# set -ue

# web

# /web:ddg_search --option adapter="echo" --option query="what is up?"
# /web:bing_search --option adapter="echo" --option query="what is up?"
# /web:google_search --option adapter="echo" --option query="what is up?"
# /web:brave_search --option adapter="echo" --option query="what is up?"

# /agent:web/search --option adapter="echo" \
#     --option query="what is up?"
/agent:web/scrape --option adapter="echo" \
    --option query="what is up?"

#
# printenv

echo "$?"
echo "*** standard/incubator tests completed ***"
exit 0
###

