<!-- /usr/bin/env ai /sh:run_task --task-name default --script -->
# TASK File Example

## Tasks

### Default

```yaml
#!/bin/bash /task:help --script
kit: task
tools:
    - name: help
      type: func
      body:
        mime-type: text/*
        script: |
            ...
```

### Build

```bash
#!/bin/bash ai /sh:bash --script
time /bin/bash ./build.sh
```

### Test

```bash
#!/bin/bash ai /sh:bash --script
go test -short ./..."
```

### All
Requires: tidy, build, test, install

```bash
#!/bin/bash
time ./test/all.sh
```

### Tidy

```bash
#!/bin/bash ai /sh:bash --script
##
go mod tidy
go fmt ./...
go vet ./...
```

### Install

```bash
#!/bin/bash ai /sh:bash --script

time CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o "$(go env GOPATH)/bin/ai" -ldflags="-w -extldflags '-static' ${CLI_FLAGS:-}" ./cmd
```

### Update

```bash
#!/bin/bash ai /sh:bash --script
go get -u ./...
```

### Clean Cache
Name: clean-cache

```bash
#!/bin/bash ai /sh:bash --script
go clean -modcache
```
