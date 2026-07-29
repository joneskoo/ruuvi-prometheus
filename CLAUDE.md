# CLAUDE.md

Guidance for working in this repository.

## Overview

A Prometheus exporter for Ruuvi Bluetooth LE sensors. It listens to Ruuvi BLE
advertisements (data formats 3, 5 and 6 — format 6 / Ruuvi Air carries air
quality), keeps the latest reading per device in Prometheus metric vectors, and
serves them over HTTP at `/metrics` (default `:9521`).

## Commands

- Build: `go build ./...`
- Test: `go test ./...`
- Vet: `go vet ./...`
- Run (needs Linux + a BLE adapter, root, and `bluetoothd` stopped):
  `sudo hciconfig hci0 down` then `sudo ./ruuvi-prometheus`. Flags: `-listen`,
  `-device` (HCI device, default `hci0`), `-debug`, `-version`.

There is no CI; run the tests locally before committing.

## Architecture

Data flows one way: BLE advertisement → decode → set metric; HTTP scrapes read
the metric vectors.

- `internal/bluetooth` — generic, Ruuvi-agnostic `Scanner`. It opens a raw HCI
  socket in `Scan`, applies the caller-supplied `filter.AdFilter`s, and
  dispatches each `host.ScanReport` to the registered handlers. `Scan` is
  hardware-bound (real HCI socket) and is not unit-tested.
- `internal/filter`, `internal/hci`, `internal/host`, `internal/logging` — the
  forked BLE stack. `internal/ruuvi` — the Ruuvi frame decoder and the
  `ruuvi.ManufacturerID` constant (the single source of truth for Ruuvi
  identity). `main` and `metrics` import these directly.
- `ruuvi-prometheus.go` — `main` is where the generic scanner and the Ruuvi
  domain are combined: `ruuviAdFilter()` builds a vendor filter from
  `ruuvi.ManufacturerID`, and `handleRuuviAdvertisement` runs `ruuvi.Decode` on
  each AD structure and calls `metrics.ObserveRuuvi`. It also starts the HTTP
  server and handles SIGINT for shutdown.
- `cmdline.go` — flag parsing into `settings`.
- `metrics/metrics.go` — `ObserveRuuvi` records one reading into `promauto`
  metric vectors (device label = MAC). A background TTL sweep (`clearExpired`,
  every minute, `ttl = 1m`) removes series for devices no longer heard from.
- `metrics/handlers.go` — `Handler` mux serving `/` (description) and `/metrics`
  (the default Prometheus registry).

## Conventions

- Metrics use `promauto` on the **default registry** (so Go/process collectors
  come for free). Follow Prometheus naming: unit suffixes, base-ish units,
  `_total` **only** on counters, and info-style metadata as an `_info` metric
  whose value is always `1` with the detail in a label (see `ruuvi_format_info`).
- Only export a value when the sensor reports it valid (`o.XxxValid()`); air
  quality readings are gated on `o.Calibrating`.
- Stale-series cleanup: every metric vector that carries a `device` label MUST
  be listed in `deviceVecs`, otherwise its series leak past the TTL. Add new
  device-labelled metrics there.
- Testing: `metrics` tests drive `ObserveRuuvi` with real captured frames (see
  the format-5/6 hex constants) and assert via `prometheus/testutil` — no BLE
  hardware needed. Tests share global metric state, so clear a device with
  `clearDevice` before observing. `main` and `bluetooth.Scan` are hardware-bound
  and intentionally not unit-tested, which caps aggregate coverage well below the
  `metrics` package's own level.

## Vendored BLE/Ruuvi stack

The BLE scanning stack and Ruuvi frame decoder are a vendored fork of
[bluewalker](https://gitlab.com/jtaimisto/bluewalker) by Jukka Taimisto,
extended here with Ruuvi data format 6 support. There is no external
`bluewalker` dependency; the only third-party runtime dependency for BLE is
`golang.org/x/sys` (raw HCI sockets).

- The whole stack lives under `internal/` (`bluetooth`, `filter`, `hci`,
  `host`, `logging`, `ruuvi`). Go's internal-package rule keeps it out of reach
  of any external importer; `main` and `metrics` (rooted at the module root)
  import the packages directly, the same way this project used the upstream
  `bluewalker` packages before the fork.
- The fork now lives in this repo and no longer tracks upstream, so edit it in
  place when it makes sense (e.g. adding Ruuvi data formats). Keep responsibilities
  split: `filter` is generic AD filtering, `bluetooth` is a Ruuvi-agnostic
  scanner, Ruuvi specifics (ids, decoding) live in `internal/ruuvi`, and the
  combination of the two lives in `main`.
- The fork carries bluewalker's 2-clause BSD license in `internal/LICENSE`.
  Keep that file, and keep the upstream copyright headers in the forked source.
