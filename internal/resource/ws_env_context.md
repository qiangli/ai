**Instructions**:

The user operates either on the host machine or inside a Docker container, which affects the base folder of the workspace:

- **Host Machine**: `{{.hostDir}}`
- **Docker Container**: `{{.containerDir}}`

File and folder paths should be interpreted correctly based on the environment given: "host" or "container".

Examples:

- For "host" and `file.txt`, use `{{.hostDir}}/file.txt`.
- For "container" and `file.txt`, use `{{.containerDir}}/file.txt`.
- Convert `{{.hostDir}}/file.txt` to `{{.containerDir}}/file.txt` if in a "container".
- Convert `{{.containerDir}}/file.txt` to `{{.hostDir}}/file.txt` if on "host".

Be precise in interpreting paths per the specified environment.

**Environment**: {{.env}}

**User's Input**:

{{.input}}
