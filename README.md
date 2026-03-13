# Server Watcher

A VPS latency monitoring tool written in Golang with SQLite and a web dashboard.

## Features

- Configurable VPS list and server port via `config.toml`.
- Password protection via `.env` file.
- Continuous concurrent PING monitoring.
- Data storage in SQLite (Min, Max, Avg, Median latency every 60s).
- Modern web dashboard with filtering and auto-refresh.

## How to Run

1.  Edit `config.toml` to add your VPS list and set the port.
2.  Create a `.env` file and set your password:
    ```bash
    echo "WATCHER_PASSWORD=your_secret_password" > .env
    ```
3.  Install dependencies:
    ```bash
    go mod tidy
    ```
4.  Build and run:
    ```bash
    ./build.sh
    ./_bin/nebula-server-watcher-your-os-arch
    ```
5.  Access the web dashboard at `http://localhost:8080` (or your configured port).

## Configuration

`config.toml`:
```toml
[server]
port = 8080

[[vps_list]]
id = "vps1"
name = "HK-GCP"
ip = "34.xx.xx.xx"
country = "HK"
remarks = "Google Cloud"
```

## Note on PING Permissions

On some Linux systems, you may need to allow the binary to send ICMP packets without root:
```bash
sudo setcap cap_net_raw=+ep nebula-server-watcher
```
Or on macOS, it should work with `pinger.SetPrivileged(false)`.
