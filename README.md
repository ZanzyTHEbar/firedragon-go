# Perception Engine

Perception Engine is a tool designed to capture and analyze multimodal data from a users environment. It is designed to be the foundation for a variety of applications, including but not limited to, human-computer interaction, human-robot interaction, and human-environment. The engine is designed to be modular, allowing for the easy addition of new data sources and analysis methods.

## Development

- [ ] Add support for config file

## Usage

> [!IMPORTANT]\
> TODO: This section of documentation is being worked on.

```bash
perception [OPTIONS] [PATH] [OPTIONAL TARGET PATH]
```

We also support the alias

```bash
pce [OPTIONS] [PATH] [OPTIONAL TARGET PATH]
```

### Options

> [!IMPORTANT]\
> TODO: This section of documentation is being worked on.

All commands have aliases.

```bash
completion    Generate the autocompletion script for the specified shell
help          Help about any command
rewind        Rewind the operations to an earlier state
backtest      Run a backtest session
```

### Arguments

> [!IMPORTANT]\
> TODO: This section of documentation is being worked on.

## Backtest Command

The `backtest` command allows you to run a backtest session by loading a specific session ID.

### Usage

```bash
perception-engine-server backtest --load session_id
```

### Options

- `--load`: Session ID to load for backtest

## Installation

> [!IMPORTANT]\
> TODO: This section of documentation is being worked on.

### From source

> [!IMPORTANT]\
> TODO: This section of documentation is being worked on.

1. Clone the repository
2. Make sure go is installed.

```bash
make build
```

Or to build and install

```bash
make build && make install
```

To test the executable

```bash
make test
```

## Development Quick Start

### Requirements

- Go 1.22 or higher
- NATs cli
- Make

The makefile is setup to automate the process of building and running the server and client. The server is a simple API server that listens for events and stores them in a database. The client is a CLI tool that sends events to the server.


### Installation

Clone this repository and navigate to the project directory.

#### Server

Like most Go projects, this project uses Makefiles to manage the build process. To get started, run the following commands:

`Run the project and run the API server in development mode`:

```bash
make dev
```

`Help`:

Show the available make commands:

```bash
make help
```

#### Client CLI

```bash
go run cmd/client.go start --nats "nats://localhost:<port>" --nats-user local --nats-pass <pass>
```

### Server CLI

```bash
go run cmd/server.go start all --nats "nats://localhost:<port>" --nats-user local --nats-pass <pass>
```

### NATs CLI

#### Sub to all events

```bash
nats sub ">" --server localhost:<port> --user local --password <pass>
```

### Data Directory Path

```text
data/<session_id>/
  ├── index.json
  ├── hid/
  │   └── hid.json
  ├── screen/
  │   └── YYYYMMDD.png
  └── transcription/
      └── <client_id>_transcript_<number>.json
```

## License

[MIT](/LICENSE)
