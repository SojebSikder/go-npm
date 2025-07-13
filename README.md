# Description

Npm like package manager created using Go.

## Install

Run

```bash
go run . install
```

## Build

```
./build.sh
```

## Supported commands

- `init` - for initialize package.json
- `install` - install packages, also support `--dev` flag
- `add` - install specific package
- `remove` - remove specific package
- `ci` - install packages from package-lock.json
- `run` - run custom scripts

## Tests

```bash
go test ./...
# with benchmark
go test ./... -bench=.
```
