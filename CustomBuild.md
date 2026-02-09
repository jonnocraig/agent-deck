# Custom Build

Build and install `agentic-deck` as a standalone binary:

```bash
cd /Users/jon_ec/code/research/agent-deck
go build -o agentic-deck ./cmd/agent-deck/
sudo cp agentic-deck /usr/local/bin/agentic-deck
```

Verify the installation:

```bash
which agentic-deck
agentic-deck --help
```

To rebuild after making changes, repeat the build and copy steps above.
