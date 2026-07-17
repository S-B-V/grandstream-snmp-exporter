# Grandstream SNMP Exporter (v2c/v3) → Prometheus

This exporter connects to a Grandstream device using SNMP v2c or v3 and exposes metrics in Prometheus text format on `/metrics`.

It collects a curated set of standard HOST-RESOURCES-MIB, IF-MIB and SNMPv2-MIB objects. For `DEVICE_TYPE=AP`, it also collects the working Grandstream AP enterprise subtree for device identity, radios and connected wireless clients.

> Strings are exported as `*_info` metrics with `value="..."` label and sample value `1`.

---

## Features

- SNMP v2c and v3 (noAuthNoPriv, authNoPriv, authPriv)
- Device type specific:

   - AP radio configuration, traffic, errors and drops
   - AP connected-client count, identity, signal and association duration

- Operationally useful standard metrics:

   - normalized CPU labels (`core="0"`, `core="1"`, ...)
   - readable interface names, state and reported speed
   - 64-bit interface traffic counters
   - storage and physical-memory utilization ratios

- Prometheus endpoint:

   - `GET /metrics`
   - `GET /healthz`

- Container-ready (Podman/Kubernetes)

- Two importable Grafana dashboards:

   - `Grafana_Dashboard.json`: system, storage, interfaces and AP wireless
   - `Grafana_Dashboard_no_interfaces.json`: compact system and AP wireless view

---

## Device-specific data

The exporter always collects the supported standard MIB objects. `DEVICE_TYPE=AP` additionally walks `1.3.6.1.4.1.42397.1.1`, which has been verified on a GWN7672.

The GCC60xx vendor subtree `1.3.6.1.4.1.12581.2` may return no objects on current GCC firmware. GCC monitoring therefore uses its extensive standard HOST-RESOURCES-MIB and IF-MIB data.

The repository includes the vendor MIB files as references, but metric names and table handling are implemented directly in the collector and do not require runtime MIB parsing.

---

## Configuration (Environment Variables)

Required:

- `DEVICE_TYPE` = `GCC|AP|L2-LITE-SWITCH|SWITCH|ROUTER|GENERIC`
- `DEVICE_IP` = device IP address
- `SNMP_VERSION` = `2c` or `3`

Optional:

- `SNMP_PORT` = `161` (default)
- `LISTEN` = `:9109` (default)

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
podman build --no-cache --platform linux/amd64 -t grandstream-snmp-exporter:latest -f Containerfile_Podman .
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

## Kubernetes (Prometheus Operator)

If you use the Prometheus Operator, the recommended integration is:

- Deployment + Service
- ServiceMonitor in the same namespace (or label-selected by Prometheus)

### ServiceMonitor (example)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: grandstream-snmp-exporter
  labels:
    release: prometheus
spec:
  selector:
    matchLabels:
      app: grandstream-snmp-exporter
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
```

> Adjust `metadata.labels.release` to match your Prometheus Helm release label if required.

---

## Kubernetes (Deployment + Service example)

### Secret example (v2c)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: snmp-secret
type: Opaque
stringData:
  community: public
```

### Deployment + Service

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grandstream-snmp-exporter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grandstream-snmp-exporter
  template:
    metadata:
      labels:
        app: grandstream-snmp-exporter
    spec:
      containers:
        - name: exporter
          image: grandstream-snmp-exporter:latest
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: 9109
          env:
            - name: LISTEN
              value: ":9109"
            - name: DEVICE_TYPE
              value: "SWITCH"
            - name: DEVICE_IP
              value: "192.0.2.12"
            - name: SNMP_VERSION
              value: "2c"
            - name: SNMP_COMMUNITY
              valueFrom:
                secretKeyRef:
                  name: snmp-secret
                  key: community
---
apiVersion: v1
kind: Service
metadata:
  name: grandstream-snmp-exporter
  labels:
    app: grandstream-snmp-exporter
spec:
  selector:
    app: grandstream-snmp-exporter
  ports:
    - name: http
      port: 9109
      targetPort: 9109
```

---

## Kubernetes CronJob pattern (optional)

Prometheus generally expects exporters to be long-running and scraped.
If you _must_ run as a CronJob, consider using a Pushgateway. You can extend the app to support `MODE=once` + `PUSHGATEWAY_URL`.
(If you want, I can add a ready-to-use Pushgateway mode.)

---

## Makefile

A Makefile is included for convenience:

- `make build` (local binary)
- `make image` (podman build)
- `make run-ap` (example run)
- `make k8s-servicemonitor` (prints ServiceMonitor YAML)

---

## Notes / Caveats

- Scrape output size depends on the number of storage entries, interfaces and AP clients reported by the device.
- AP client identity fields are metric labels. On large wireless deployments, consider disabling or removing per-client metrics to limit Prometheus cardinality.
- Interface counters use 64-bit IF-MIB objects when available. Some virtual interfaces do not report a usable link speed, so interface-utilization calculations omit them.
- Strings are exported as `*_info` metrics (`value="..."`) with sample `1`. If you need to avoid high-cardinality string labels, change the exporter to hash/truncate values.

---

## Troubleshooting

### No metrics / exporter_up = 0

- Check SNMP reachability from the container/pod (network policies, firewall).
- Confirm correct SNMP version + credentials.
- Confirm `DEVICE_TYPE` matches the device you’re scraping.
- Check whether the device exposes HOST-RESOURCES-MIB and IF-MIB.
- For AP wireless metrics, confirm the device responds under `1.3.6.1.4.1.42397.1.1` and uses `DEVICE_TYPE=AP`.
