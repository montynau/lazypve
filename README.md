# lazypve

A fast, interactive terminal UI for keeping an eye on Proxmox VE — in the spirit of [lazygit](https://github.com/jesseduffield/lazygit) and [lazydocker](https://github.com/jesseduffield/lazydocker), but for your cluster.

## Why

Clicking through the Proxmox web UI just to check whether a node is under load, or which VM ate all the RAM, gets old fast. lazypve polls the Proxmox API in the background and keeps a live table of node status, CPU, and memory usage right in your terminal — no browser tab required.

It's read-only by design for now: lazypve won't start, stop, or touch anything on your cluster. It just watches.

## Status

Live node status, a full VM/LXC listing (CPU%, memory, live network throughput, uptime), interactive drill-down navigation, sortable columns, and multi-cluster support (with a warning if one cluster drops out) are all working. See [Roadmap](#roadmap) for what's tracked next.

## Requirements

- A Proxmox VE host you can reach over the network
- An API token (see [Setup](#setup) below)
- [Go](https://go.dev/) 1.26+ if building from source

## Install

```sh
git clone git@github.com:MontyNau/lazypve.git
cd lazypve
go build -o lazypve ./cmd/lazypve
```

## Setup

lazypve authenticates with a Proxmox API token rather than a username/password, so it can run with least-privilege, read-only access.

1. In the Proxmox web UI: **Datacenter → Permissions → API Tokens → Add**. Pick a user (`root@pam` is fine), give the token an ID (e.g. `lazypve`), keep **Privilege Separation** checked, and save. Copy the **Secret** shown — it's only displayed once.
2. **Datacenter → Permissions → Add → API Token Permission**: Path `/`, select the token you just created, Role **`PVEAuditor`** (a built-in read-only role), keep **Propagate** checked.
3. Copy `.env.example` to `.env` and fill in your values:

   ```sh
   cp .env.example .env
   ```

   ```env
   LAZYPVE_HOST=https://<your-pve-ip>:8006
   LAZYPVE_TOKEN_ID=root@pam!lazypve
   LAZYPVE_TOKEN_SECRET=<the secret from step 1>
   LAZYPVE_INSECURE_SKIP_VERIFY=true   # Proxmox uses a self-signed cert by default
   ```

### Multiple clusters

If you manage more than one Proxmox cluster, repeat the token setup above for each, then list them in `.env` as a numbered set instead of the plain `LAZYPVE_*` vars:

```env
LAZYPVE_CLUSTER1_NAME=homelab
LAZYPVE_CLUSTER1_HOST=https://192.168.1.10:8006
LAZYPVE_CLUSTER1_TOKEN_ID=root@pam!lazypve
LAZYPVE_CLUSTER1_TOKEN_SECRET=...
LAZYPVE_CLUSTER1_INSECURE_SKIP_VERIFY=true

LAZYPVE_CLUSTER2_NAME=work
LAZYPVE_CLUSTER2_HOST=https://10.0.0.5:8006
LAZYPVE_CLUSTER2_TOKEN_ID=root@pam!lazypve
LAZYPVE_CLUSTER2_TOKEN_SECRET=...
LAZYPVE_CLUSTER2_INSECURE_SKIP_VERIFY=true
```

Add `LAZYPVE_CLUSTER3_*`, etc. for more — lazypve reads `LAZYPVE_CLUSTER1_*`, `LAZYPVE_CLUSTER2_*`, ... and stops at the first missing number. Every table then gains a `CLUSTER` column, and drill-down filters by cluster + node together (so two clusters can safely have a node with the same name).

## Usage

```sh
go run ./cmd/lazypve
# or, if you built a binary:
./lazypve
```

`tab` switches focus between the nodes and guests tables, `↑`/`↓` (or `j`/`k`) move the selection, `enter` drills into the selected node's guests, `q` quits.

`]`/`[` sort the focused table by the next/previous column — press the same key again to reverse direction before it moves on to the next column. `esc` resets whichever of these is active on the focused table: the drill-down filter, its sort, or both at once.

## Roadmap

- [x] Node status, CPU, memory (live)
- [x] VM and LXC listing per node
- [x] Drill-down / navigation between nodes and guests
- [x] Disk usage and live network throughput (VM/LXC)
- [x] Multi-cluster support, with a warning if a cluster drops out
- [x] Sortable columns

Start/stop/restart control is intentionally out of scope until the monitoring core is solid.

## License

[MIT](LICENSE)
