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

## Example: SNMP v2c

```bash
docker run --rm -p 9109:9109 \
  -e DEVICE_TYPE=grandstream \
  -e DEVICE_IP=192.0.2.10 \
  -e SNMP_VERSION=2c \
  -e SNMP_COMMUNITY=public \
  ghcr.io/s-b-v/grandstream-snmp-exporter:latest
```

## Example: SNMP v3 authPriv

```bash
docker run --rm -p 9109:9109 \
  -e DEVICE_TYPE=grandstream \
  -e DEVICE_IP=192.0.2.10 \
  -e SNMP_VERSION=3 \
  -e SNMP_SECURITY_LEVEL=authPriv \
  -e SNMP_USERNAME=myuser \
  -e SNMP_AUTH_PROTOCOL=SHA-256 \
  -e SNMP_AUTH_PASSPHRASE='my-auth-passphrase' \
  -e SNMP_PRIV_PROTOCOL=AES \
  -e SNMP_PRIV_PASSPHRASE='my-privacy-passphrase' \
  ghcr.io/s-b-v/grandstream-snmp-exporter:latest
```
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
podman build --no-cache --platform linux/amd64 -t grandstream-snmp-exporter:latest -f dockerfile .

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
