# Grandstream Grafana dashboards

Import the three JSON dashboards through Grafana > Dashboards > New > Import. Map `${DS_PROMETHEUS}` to your existing Prometheus data source.

- `Grafana_Dashboard.json`: general health + interfaces, UID `grandstream-full`.
- `Grafana_Dashboard_no_interfaces.json`: compact health dashboard, UID `grandstream-compact`.
- `Grafana_Dashboard_wifi_clients.json`: wireless inventory, AP/SSID/client filter, and time-history, UID `grandstream-wifi-clients`.

**Requirements:** `DEVICE_TYPE=AP` on all three AP exporters, `grandstream_ap_client_info`, `_signal`, `_association_seconds`, Prometheus 30s scraping and Grafana >=10. Existing dashboard UIDs may overwrite prior versions on import; export backups first.

**Limitations:** client association and roaming history is sampled, not precise event logging. Signal is a raw positive integer in the user's SNMP walk, not verified dBm. Instant query inventory depends on successful scrapes. Missing hostnames appear blank. Per-client labels may increase time-series cardinality; no third-party cloud or SQL storage is used.
