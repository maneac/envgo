# Envgo

[![License: MIT](https://img.shields.io/github/license/maneac/envgo)](https://opensource.org/licenses/MIT)
[![Go workflow](https://github.com/maneac/envgo/actions/workflows/golang.yml/badge.svg)](https://github.com/maneac/envgo/actions/workflows/golang.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/maneac/envgo)](https://go.dev)
[![Latest release](https://img.shields.io/github/v/release/maneac/envgo)](https://github.com/maneac/envgo/releases/latest)

Small utility to allow Go code to be executed in a shell-like fashion. Designed to emulate the scripting functionality offered
by Python and Ruby.

## Features

- Invocation using UNIX shebang (`#!`) notation
- Supports third-party dependencies

## Requirements

- A working Go v1.16+ install, present in your `PATH`

## Installation

```sh
go install github.com/maneac/envgo@latest
```

## Usage

```sh
envgo [-v] FILEPATH
```

- `FILEPATH` is the path to the `main.go`-equivalent contents to run. See [Example](#Example) below for more details.
- `-v` enables verbose logging of the pre-execution script processing stages

## Example

More examples can be found in the `examples` directory. The examples below are using
the contents of `examples/readme-contents`.

### Calling `envgo` directly

1. Create a file containing the contents to run:

    ```sh
    echo $FILE_CONTENTS > "script"
    ```

2. Pass the filepath to `envgo` as the single parameter:

    ```sh
    envgo ./script
    # Output:
    # INFO[0000] running script
    # foo
    # INFO[0001] input:foo
    ```

### Standalone Script

1. Create a file containing the contents to run, with a shebang line at the start:

    ```sh
    echo "#!/usr/bin/env envgo" > script
    echo $FILE_CONTENTS >> script
    ```

    Contents of `script`:

    ```go
    #!/usr/bin/env envgo
    package main
    ...
    ```

2. Make the file executable:

    ```sh
    chmod +x ./script
    ```

3. Run the file:

    ```sh
    ./script
    # Output:
    # INFO[0000] running script
    # foo
    # INFO[0001] input:foo
    ```

### Verbose Standalone Script

To enable verbose output for `envgo` within a standalone script, follow the steps of [Standalone Script](#Standalone-Script), with the following shebang line:

```sh
#!/usr/bin/env -S envgo -v
```

## Approach

All Go code requires compilaton before it can be executed. To enable this, `envgo` creates a temporary compilation directory, under `$TMPDIR/envgo`, and copies the contents of the supplied file to it as a `main.go`. If a `main.go` already exists in the directory, an MD5 checksum of its contents is first compared to what would be written, to avoid unnecessary copies.

Once the file contents are copied, a Go module is initialised in the temporary compilation directory, using the extensionless name of the supplied
file as the module name. The module is then tidied and built, before being executed.

Each temporary directory is named using the MD5 checksum of the absolute filepath to the script. This, in combination with the content check, allows
a compiled script binary to be re-used if the script contents have not changed.

Example temporary compilation directory name: `/tmp/envgo/6578616d706c65732f74686972642d70617274792d7061636b61676573d41d8cd98f00b204e9800998ecf8427e`

Example temporary compilation directory contents:

- `go.mod` (result of internal `go mod init` call)
- `go.sum` (result of internal `go mod tidy` call)
- `main.go` (contents of the supplied script)
- `third-party-packages` (built binary)
