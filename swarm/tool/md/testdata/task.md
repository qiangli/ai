<!-- /usr/bin/env ai /sh:run_task --task-name default --script -->
# TASK File Example

## Tasks

### Default

```yaml
#!/usr/bin/env ai /task:help --script
kit: task
tools:
    - name: help
      type: func
      body:
        mime-type: text/*
        script: |
          # Build DHNT.io AI
          TODO
```

### Build

```bash
#!/usr/bin/env ai /sh:bash --script
time /bin/bash ./build.sh
```

### Test

```bash
#!/usr/bin/env ai /sh:bash --script
go test -short ./..."
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
#!/usr/bin/env ai /sh:bash --script
time ./test/all.sh
```

### Tidy

```bash
#!/usr/bin/env ai /sh:bash --script
##
go mod tidy
go fmt ./...
go vet ./...
```

### Install

```bash
#!/usr/bin/env ai /sh:bash --script
time CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o "$(go env GOPATH)/bin/ai" -ldflags="-w -extldflags '-static' ${CLI_FLAGS:-}" ./cmd
```

### Update

Update all dependencies

```bash
#!/usr/bin/env ai /sh:bash --script
go get -u ./...
```

### Clean Cache

```bash
#!/usr/bin/env ai /sh:bash --script
go clean -modcache
```
