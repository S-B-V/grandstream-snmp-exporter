# Grandstream SNMP Exporter (v2c/v3) -> Prometheus

This exporter connects to a Grandstream device using SNMP v2c or v3 and exposes metrics in Prometheus text format on `/metrics`.

## Endpoints

- `/metrics` - Prometheus metrics
- `/healthz` - simple HTTP health check

## Environment variables

### Common

- `DEVICE_TYPE` - logical device type label
- `DEVICE_IP` - IP address or hostname of the target device
- `SNMP_VERSION` - `2c` or `3`
- `SNMP_PORT` - optional, default `161`
- `LISTEN` - optional HTTP listen address, default `:9109`

### SNMP v2c

- `SNMP_COMMUNITY`

### SNMP v3

- `SNMP_SECURITY_LEVEL` - one of:
  - `noAuthNoPriv`
  - `authNoPriv`
  - `authPriv`
- `SNMP_USERNAME`
- `SNMP_AUTH_PROTOCOL` - optional depending on security level:
  - `MD5`
  - `SHA`
  - `SHA-224`
  - `SHA-256`
  - `SHA-384`
  - `SHA-512`
- `SNMP_AUTH_PASSPHRASE`
- `SNMP_PRIV_PROTOCOL` - optional depending on security level:
  - `DES`
  - `AES`
  - `AES-192`
  - `AES-256`
- `SNMP_PRIV_PASSPHRASE`

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


## Build locally

```bash
go mod tidy
go build ./...
```

---

## Build (Podman)

From repo root:

```bash
podman build --no-cache --platform linux/amd64 -t grandstream-snmp-exporter:latest -f podmanfile .

```
---

## Notes on behaviour

- `grandstream_exporter_up{device_ip="..."}` is `1` when the exporter could create a client and connect successfully during the current scrape.
- If a scrape fails, the metric becomes `0` for that scrape.
- The next scrape retries with a completely new SNMP client.
- This design is especially useful for SNMPv3, where stale session state can otherwise cause repeated failures after interruptions.

## Files changed for the reliability fix

- `cmd/exporter/main.go`
- `internal/exporter/collector.go`
- `internal/snmp/snmp.go`
- `go.mod`

## Manual replacement

Replace the corresponding files in your repository with the versions provided here, then run:

```bash
go mod tidy
go build ./...
```
