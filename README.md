# Description

Npm like package manager created using Go.

## Features

- Initialize a new project with `package.json`
- Install dependencies and devDependencies
- Add or remove specific packages
- Lock dependencies with `package-lock.json`
- Install from lock file for reproducible builds
- Run custom scripts defined in `package.json`
- Create executable links for package binaries in `node_modules/.bin`
- Cross-platform support (Windows, Linux, macOS)

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
# with coverage
go test -cover ./...
```
