Task: Based on the provided input, please determine the appropriate workspace base and form a final response.

Input: {{.input}}

Expected Response Format:

Your response should be a valid JSON object without additional text or formatting.
Ensure the response contains the following keys:
"workspace_base": The root directory path.
"is_valid": A boolean indicating if the input is valid.
"exist": A boolean indicating if the workspace exists.
"reason": Explanation for unsuitability, or leave empty if the input is suitable.
Output Example:

{
  "workspace_base": "<root directory path>",
  "is_valid": true/false,
  "exist": true/false,
  "reason": "Provide a reason if applicable, otherwise leave empty"
}
