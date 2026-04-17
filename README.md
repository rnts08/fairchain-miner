# Fairchain Standalone Miner

High performance standalone miner for the Fairchain network. Supports solo mining via RPC and offline simulation mode.

## Features

- Solo mining logic for direct connection to full node via RPC
- Offline simulation mode for hashrate and TUI testing
- Multi-threaded execution using all available CPU cores
- Hashrate monitor with EWMA average reporting
- CPU power limiter (throttle intensity from 1-100%)
- Interactive TUI dashboard with real-time updates
- Algorithm support: sha256d, scrypt, argon2id, sha256mem

---

## Build Instructions

### Requirements

- Go 1.25 or newer
- Make

### Build

```bash
# Build both CLI and TUI versions
make all

# Build CLI version only
make cli

# Build TUI version only
make tui
```

Binaries:

- fairchain-miner: Standard CLI miner
- fairchain-miner-tui: Terminal UI miner

### Configuration File

The miner supports a `config.toml` file for persistent settings. To use it:

1. Copy `config.example.toml` to `config.toml`.
2. Edit the values to match your node setup.

Command-line flags will always override values found in the configuration file.

### Install

```bash
sudo make install
```

---

## Usage

### CLI Miner

```bash
# Solo mining
./fairchain-miner --rpc http://127.0.0.1:19335

# Configuration with 8 workers at 75% power
./fairchain-miner --rpc http://127.0.0.1:19335 --workers 8 --power 75
```

### TUI Miner

```bash
./fairchain-miner-tui --rpc http://127.0.0.1:19335
```

### Simulation Mode

Use the -t flag to launch the miner in offline simulation mode. This allows testing of the TUI and hardware performance without a network connection.

```bash
./fairchain-miner-tui -t
```

---

## Command Line Options

| Option | Default | Description |
|--------|---------|-------------|
| --rpc | <http://127.0.0.1:19335> | Full node RPC address |
| --workers | Num CPU | Number of mining threads |
| --power | 100 | CPU power limit (1-100) |
| -t | false | Run in offline simulation mode |
| --help | | Show help |

---

## Terminal UI

The TUI miner provides a terminal interface including:

- Header: Displays hashrate, shares, and current power usage.
- Graph:
  - Resolution based on terminal width.
  - Y-axis labels for MAX/MIN values.
  - Share markers (S) on the timeline showing mining success.
- Controls:
  - Navigation: Arrow keys or Tab to move between buttons.
  - Activation: Enter or Space to trigger actions.
  - Buttons: Adjust Workers (+/-), Power (+/-), and Pause/Resume.
- Log View: Full-width scrollable log for events with timestamps.
- Footer: Persistent help keymap at bottom.

Keymap:

- Tab / Arrows: Navigation
- Enter / Space: Execute action
- p: Pause/Resume mining
- +/-: Adjust workers
- [/]: Adjust power
- q: Quit

---

## Support

If you find this project useful, you can support development at these addresses:

- BTC: bc1qkmzc6d49fl0edyeynezwlrfqv486nmk6p5pmta
- ETH: 0xC13D012CdAae7978CAa0Ef5B1E30ac6e65e6b17F
- LTC: ltc1q0ahxru7nwgey64agffr7x89swekj7sz8stqc6x
- SOL: HB2o6q6vsW5796U5y7NxNqA7vYZW1vuQjpAHDo7FAMG8
- XRP: rUW7Q64vR4PwDM3F27etd6ipxK8MtuxsFs

---

## License

See LICENSE.md for details.
