#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script
set -ue

echo "# Flow Test"

# actions in valid JSON array format
/flow:sequence --actions '["sh:pwd","sh:format"]' --template "data:,***got from 'pwd': {{.result}}***\n"

# simple comma sepaated list also works 
/flow:choice --actions "[sh:pwd,sh:pass]"

/flow:parallel --actions "[sh:pwd,sh:pass]"

/flow:loop --actions "[sh:pass]" --max-iteration 3 --sleep 1s --report "Runing in a loop"

/flow:fallback --actions "[sh:fail,sh:pass]" 

# chain with timeout/backoff using the "alias" action
/flow:chain --chain "[sh:timeout,sh:backoff,alias:cmd]" \
    --option cmd="/flow:choice --actions=[sh:pass,sh:fail]" \
    --option duration="60s" \
    --option report="sh:fail was called"

#
echo "$?"
echo "*** 🎉 flow tests completed ***"
exit 0
#
