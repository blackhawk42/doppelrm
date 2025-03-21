# doppelrm

A companion utility to [doppel](https://github.com/blackhawk42/doppel). It takes
doppel's output and starts a TUI interface to make deleting repeated files easier.

TUI made using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework.

# Build

To build the main executable:

```shell
go build cmd/doppelrm
```

# Usage

```shell
./doppelrm DOPPEL_FILE
```

`DOPPEL_FILE` may also be `-` for stdin, to pipe doppel's output.

Form more help, run:

```shell
./doppelrm -h
```