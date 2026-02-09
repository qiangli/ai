<!-- /usr/bin/env ai /sh:run_task --task default --script -->
# TASK File Example

## Tasks

### Default

```yaml
#!/dev:help
kit: dev
tools:
  - name: "help"
    display: "AI Dev Build Help"
    description: |
      Steps for building the 'ai' binary
    type: "func"
    parameters: {}
    body: 
      mime_type: "text/markdown"
      script: |
        # AI Build
        ## Tidy
          ai /sh:run_task --task tidy --taskfile ./taskfile.md
        ## Test
          ai /sh:run_task --task test --taskfile ./taskfile.md
        ## Build
          ai /sh:run_task --task build --taskfile ./taskfile.md
        ## Install
          ai /sh:run_task --task install --taskfile ./taskfile.md
        ## All
          Run tidy, build, test, install, and test/all.sh:
          ai /sh:run_task --task all --taskfile ./taskfile.md
        
        Tip: add an alias to your system, e.g.:
        
        ~~~bash
        alias dev="ai /sh:run_task --taskfile ./taskfile.md --task"
        ~~~
    arguments:
      log_level: "quiet"
##
```

### Build

```bash
set -xe
time /bin/bash ./build.sh
echo "EXIT STATUS: $?"
```

### Test

```bash
set -xe
go test -short ./...
echo "EXIT STATUS: $?"
```

### All

Buid, install, unit tests, and bash integration tests

---
dependencies:
  - tidy
  - build
  - test
  - install
---

```bash
set -xe
time ./test/all.sh
echo "EXIT STATUS: $?"
```


### Test Shell

Run bash integration tests

```bash
set -xe
time ./test/all.sh
echo "EXIT STATUS: $?"
```

### Tidy

```bash
set -xe
go mod tidy
go fmt ./...
go vet ./...
echo "EXIT STATUS: $?"
```

### Install

```bash
set -xe
time CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o "$(go env GOPATH)/bin/ai" -ldflags="-w -extldflags '-static' ${CLI_FLAGS:-}" ./cmd
echo "EXIT STATUS: $?"
```

### Update

Update all dependencies

```bash
set -xe
go get -u ./...
echo "EXIT STATUS: $?"
```

### Clean Cache

```bash
set -xe
go clean -modcache
echo "EXIT STATUS: $?"
```
