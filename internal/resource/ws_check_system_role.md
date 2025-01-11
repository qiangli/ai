Analyze the user's input to detect if a workspace base directory is mentioned or implied.

Response Based on Analysis:

If a base directory is implied or explicitly mentioned, proceed with evaluating its suitability as per the following criteria. If no base directory is detected, respond with the message: "Please specify a workspace base directory."

1. **Directory Status**:
   - **Non-Existent**: If the directory does not exist, it is suitable. No further checks are needed.
   - **Empty**: If the directory exists and is empty, it is suitable. No additional actions are required.
   - **Non-Empty**: If the directory has contents, proceed to verify:

2. **Git Verification for Non-Empty Directory**:
   - Check if the directory is a Git repository.
   - Ensure the repository has a clean state: no uncommitted changes and it is up-to-date with the remote.

### Steps for Confirmation

- Confirm existence and contents of the directory.
- For non-empty directories, verify Git status to determine suitability.
