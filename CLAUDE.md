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

- `bluetooth/bluetooth.go` — `Scanner` wraps
  [bluewalker](https://gitlab.com/jtaimisto/bluewalker). It filters on the Ruuvi
  vendor id, opens a raw HCI socket in `Scan`, and dispatches each `ScanReport`
  to the registered handlers. `Scan` is hardware-bound (real HCI socket) and is
  not unit-tested.
- `ruuvi-prometheus.go` — `main` wires the scanner to
  `handleRuuviAdvertisement`, which runs `ruuvi.Decode` on each AD structure and
  calls `metrics.ObserveRuuvi`. It also starts the HTTP server and handles
  SIGINT for shutdown.
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

## Planned work

- **Extract the Bluetooth/Ruuvi parsing into a new, self-owned library and drop
  the `bluewalker` dependency.** Agreed with bluewalker's author (jtaimisto): the
  relevant BLE and Ruuvi-parsing code will be extracted into a new module owned
  by this project, carrying bluewalker's existing license. Once that module
  exists, plan which code moves out of `ruuvi-prometheus` into it — the new
  module should expose `bluetooth` and `ruuvi` as separate packages — leaving
  `ruuvi-prometheus` a thin exporter on top. Candidates to move: the `bluetooth/`
  scanner wrapper and the frame-decoding currently provided by `bluewalker/ruuvi`
  (with `handleRuuviAdvertisement` reduced to glue).
