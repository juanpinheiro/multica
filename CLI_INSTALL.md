# Multica CLI — Installation Guide

Build the Multica CLI from source on your local machine.

## Prerequisites

- Go 1.26+
- Git

## Installation

### Step 1: Clone the repository

```bash
git clone https://github.com/multica-ai/multica.git
cd multica
```

### Step 2: Build the CLI

Using the Makefile (recommended):

```bash
make build
cp server/bin/multica /usr/local/bin/multica
```

Or directly with Go:

```bash
cd server
go build -o ../bin/multica ./cmd/multica
cp ../bin/multica /usr/local/bin/multica
```

### Step 3: Verify installation

```bash
multica version
```

Expected output: `multica v0.x.x (commit: ...)`

**If installation fails:**
- Ensure Go 1.26+ is installed: `go version`
- Ensure `/usr/local/bin` is in your `$PATH`
- On Linux, you may need to make the binary executable: `chmod +x /usr/local/bin/multica`
- If you prefer a user-local install: `cp ../bin/multica ~/.local/bin/multica` and add `~/.local/bin` to your `$PATH`

---

## Next Steps

Once installed, set up the daemon:

```bash
multica setup
```

This configures the CLI, opens your browser for authentication, and starts the daemon in one command.

For self-hosted servers:

```bash
multica setup self-host --server-url https://your-server.com
```

Check daemon status:

```bash
multica daemon status
```

For more information, see [CLI_AND_DAEMON.md](CLI_AND_DAEMON.md).
