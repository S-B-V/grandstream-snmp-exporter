# Grandstream SNMP Exporter (v2c/v3) → Prometheus

This exporter connects to a Grandstream device using SNMP v2c or v3, collects the device-specific enterprise subtree, and exposes metrics in Prometheus text format on `/metrics`.

It also uses Grandstream MIBs shipped inside the container image to:

- resolve numeric OIDs to stable metric names (`grandstream_<mibSymbol>`)
- derive meaningful table index labels from the MIB `INDEX { ... }` definitions (e.g. `ifIndex`, `vlanIndex`, ...)

> Strings are exported as `*_info` metrics with `value="..."` label and sample value `1`.

---

## Features

- SNMP v2c and v3 (noAuthNoPriv, authNoPriv, authPriv)
- Device type specific:

   - loads only the required MIB(s) for that device type
   - walks only the correct subtree for that device type

- Prometheus endpoint:

   - `GET /metrics`
   - `GET /healthz`

- Container-ready (Podman/Kubernetes)

---

## Kubernetes (Deployment + Service example)

- See k8s folder for working examples. 

---

## Device type → MIB mapping (inside the container)

⚠️ It seems that Grandstream devices are not responding to their MIBs... but I keept them, just in case they may work in future...

MIBs are included under `/mibs` and the exporter loads:

- `DEVICE_TYPE=AP`

   - `GRANDSTREAM-GWN-ROOT-MIB.txt`
   - `GRANDSTREAM-GWN-PRODUCTS-AP-MIB.txt`

- `DEVICE_TYPE=GCC`

   - `GS-GCC60XX-SNMP-MIB-V1.0.txt`

- `DEVICE_TYPE=L2-LITE-SWITCH`

   - `GRANDSTREAM-GWN-ROOT-MIB.txt`
   - `GRANDSTREAM-GWN-PRODUCTS-L2-LITE-SWITCH-MIB.txt`

- `DEVICE_TYPE=GENERIC`

   - `GRANDSTREAM-GWN-ROOT-MIB.txt`
   - `GRANDSTREAM-GWN-PRODUCTS-MIB.txt`

- `DEVICE_TYPE=ROUTER`

   - `GRANDSTREAM-GWN-ROOT-MIB.txt`
   - `GRANDSTREAM-GWN-PRODUCTS-ROUTER-MIB.txt`

- `DEVICE_TYPE=SWITCH`

   - `GRANDSTREAM-GWN-ROOT-MIB.txt`
   - `GRANDSTREAM-GWN-PRODUCTS-SWITCH-MIB.txt`

---

## Configuration (Environment Variables)

Required:

- `DEVICE_TYPE` = `GCC|AP|L2-LITE-SWITCH|SWITCH|ROUTER|GENERIC`
- `DEVICE_IP` = device IP address
- `SNMP_VERSION` = `2c` or `3`

Optional:

- `SNMP_PORT` = `161` (default)
- `LISTEN` = `:9109` (default)
- `MIB_DIR` = `/mibs` (default; container already sets this)

### SNMP v2c

Required:

- `SNMP_COMMUNITY`

### SNMP v3

Required:

- `SNMP_SECURITY_LEVEL` = `noAuthNoPriv|authNoPriv|authPriv`
- `SNMP_USERNAME`

Conditional:

- If `authNoPriv` or `authPriv`:

   - `SNMP_AUTH_PROTOCOL` = `MD5|SHA|SHA-224|SHA-256|SHA-384|SHA-512`
   - `SNMP_AUTH_PASSPHRASE`

- If `authPriv`:

   - `SNMP_PRIV_PROTOCOL` = `DES|AES|AES-192|AES-256`
   - `SNMP_PRIV_PASSPHRASE`

---

## Build (Podman)

From repo root:

```bash
podman build --no-cache --platform linux/amd64 -t grandstream-snmp-exporter:latest -f dockerfile .
```

---

## Run examples (Podman)

### AP (SNMP v2c)

```bash
podman run --rm -p 9109:9109 \
  -e DEVICE_TYPE=AP \
  -e DEVICE_IP=192.0.2.10 \
  -e SNMP_VERSION=2c \
  -e SNMP_COMMUNITY=public \
  grandstream-snmp-exporter:latest
```

### Router (SNMP v3 authPriv)

```bash
podman run --rm -p 9109:9109 \
  -e DEVICE_TYPE=ROUTER \
  -e DEVICE_IP=192.0.2.11 \
  -e SNMP_VERSION=3 \
  -e SNMP_SECURITY_LEVEL=authPriv \
  -e SNMP_USERNAME=myuser \
  -e SNMP_AUTH_PROTOCOL=SHA-256 \
  -e SNMP_AUTH_PASSPHRASE='auth-pass' \
  -e SNMP_PRIV_PROTOCOL=AES \
  -e SNMP_PRIV_PASSPHRASE='priv-pass' \
  grandstream-snmp-exporter:latest
```

### Switch (SNMP v2c)

```bash
podman run --rm -p 9109:9109 \
  -e DEVICE_TYPE=SWITCH \
  -e DEVICE_IP=192.0.2.12 \
  -e SNMP_VERSION=2c \
  -e SNMP_COMMUNITY=public \
  grandstream-snmp-exporter:latest
```

### GCC (SNMP v3 authNoPriv)

```bash
podman run --rm -p 9109:9109 \
  -e DEVICE_TYPE=GCC \
  -e DEVICE_IP=192.0.2.20 \
  -e SNMP_VERSION=3 \
  -e SNMP_SECURITY_LEVEL=authNoPriv \
  -e SNMP_USERNAME=myuser \
  -e SNMP_AUTH_PROTOCOL=SHA-256 \
  -e SNMP_AUTH_PASSPHRASE='auth-pass' \
  grandstream-snmp-exporter:latest
```

Check:

- http://localhost:9109/healthz
- http://localhost:9109/metrics


---

## Makefile

A Makefile is included for convenience:

- `make build` (local binary)
- `make image` (podman build)
- `make run-ap` (example run)
- `make k8s-servicemonitor` (prints ServiceMonitor YAML)

---

## Notes / Caveats

- Scrape output size depends on how many OIDs exist under the walked subtree.
- Table index labels are derived from MIB `INDEX { ... }`. This implementation supports the common case where index values are integer arcs.
- Strings are exported as `*_info` metrics (`value="..."`) with sample `1`. If you need to avoid high-cardinality string labels, change the exporter to hash/truncate values.

---

## Troubleshooting

### No metrics / exporter_up = 0

- Check SNMP reachability from the container/pod (network policies, firewall).
- Confirm correct SNMP version + credentials.
- Confirm `DEVICE_TYPE` matches the device you’re scraping.
- Confirm MIB files exist in the image under `/mibs`.

### Metrics show `grandstream_oid{oid="..."}`

- That means the OID was not resolved by the loaded MIB(s).
- Ensure the right `DEVICE_TYPE` is set (so the correct MIB is loaded).
- If needed, add additional dependency MIBs to `/mibs` and extend the loader.
