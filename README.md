# ruuvi-prometheus exporter

This is a simple Prometheus exporter that exports metrics for
Ruuvi version 3 data format Bluetooth LE advertisements.

## Usage

I use ruuvi exporter with Alpine Linux myself and it’s on
[Alpine Linux repository in edge/testing].

To use this binary release, install Alpine edge and enable the testing
repository and apk add ruuvi-prometheus.
Enable service to start at booth with `rc-update add ruuvi-prometheus`.

You will need to uncomment "rpi bluetooth" in /etc/mdev.conf for
Raspberry Pi Bluetooth to work.

The service binds listen address to `:9521` by default.

For usage with Grafana, see [grafana-example-dashboard.json](./grafana-example-dashboard.json).

To keep the exporter stateless (I run it on a headless read-only Raspberry),
it is better to add the logical names for sensors in Prometheus configuration.
For example:

```yaml
  - job_name: 'ruuvi-prometheus'
    static_configs:
      - targets: ['ruuvi-prometheus:9521']
    metric_relabel_configs:
      # ... id and name relabel config for each sensor
      - source_labels: ['device']
        target_label: 'id'
        regex: 'e7:37:3b:37:d9:74'
        replacement: '2'
      - source_labels: ['id']
        target_label: 'name'
        regex: '2'
        replacement: 'garage'
      # Map location based on id
      - source_labels: ['id']
        regex: '2|3|5|6|7|9'
        target_label: 'location'
        replacement: 'indoors'
      - source_labels: ['id']
        regex: '1|4'
        target_label: 'location'
        replacement: 'outdoors'
```

## Further development

Ideally I would like to run this using [gokrazy] instead, but
my current setup requires WiFi with WPA2 and Raspberry Pi Bluetooth
which are not supported yet. Once those blockers are solved, gokrazy
can create Raspberry appliance image without any C code besides Linux.

[Alpine Linux repository in edge/testing]: https://pkgs.alpinelinux.org/packages?name=ruuvi-prometheus&arch=armhf
[gokrazy]: https://gokrazy.org/

## Exported metrics

<dl>
  <dt>ruuvi_acceleration_g</dt>
  <dd>Ruuvi tag sensor acceleration X/Y/Z</dd>

  <dt>ruuvi_battery_volts</dt>
  <dd>Ruuvi tag battery voltage</dd>

  <dt>ruuvi_frames_total</dt>
  <dd>Total Ruuvi frames received</dd>

  <dt>ruuvi_humidity_ratio</dt>
  <dd>Ruuvi tag sensor relative humidity</dd>

  <dt>ruuvi_pressure_hpa</dt>
  <dd>Ruuvi tag sensor air pressure</dd>

  <dt>ruuvi_rssi_dbm</dt>
  <dd>Ruuvi tag received signal strength RSSI</dd>

  <dt>ruuvi_temperature_celsius</dt>
  <dd>Ruuvi tag sensor temperature</dd>

  <dt>ruuvi_format</dt>
  <dd>Ruuvi frame format version (e.g. 3 or 5)</dd>

  <dt>ruuvi_movecount_total</dt>
  <dd>Ruuvi movement counter</dd>

  <dt>ruuvi_seqno_current</dt>
  <dd>Ruuvi frame sequence number</dd>

  <dt>ruuvi_txpower_dbm</dt>
  <dd>Ruuvi transmit power in dBm</dd>
</dl>

## System requirements

* Linux
* Bluetooth LE; bluetoothd must not be running.

[bluewalker]: https://gitlab.com/jtaimisto/bluewalker/