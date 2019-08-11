# ruuvi-prometheus exporter

This is a simple Prometheus exporter that exports metrics for
Ruuvi version 3 data format Bluetooth LE advertisements.

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