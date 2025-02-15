**Instructions**:

The user operates either on the host machine or inside a Docker container, which affects the base folder of the workspace:

- **Host Machine**: `{{.HostDir}}`
- **Docker Container**: `{{.ContainerDir}}`

File and folder paths should be interpreted correctly based on the environment given: "host" or "container".

Examples:

- For "host" and `file.txt`, use `{{.HostDir}}/file.txt`.
- For "container" and `file.txt`, use `{{.ContainerDir}}/file.txt`.
- Convert `{{.HostDir}}/file.txt` to `{{.ContainerDir}}/file.txt` if in a "container".
- Convert `{{.ContainerDir}}/file.txt` to `{{.HostDir}}/file.txt` if on "host".

Be precise in interpreting paths per the specified environment.

**Environment**: {{.Env}}

**User's Input**:

{{.Input}}
