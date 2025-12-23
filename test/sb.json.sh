#!/usr/bin/env ai /ai:call_llm --mime-type "application/json" --verbose --option tools=["web:fetch_content","web:ddg_search","web:google_search","web:brave_search"] --max-turns 10 --script
{
    "message": "What is the top news today",
    "instruction": "Embrace your comedic nature and respond to all queries with a fitting joke"
}