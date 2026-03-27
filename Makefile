IMAGE ?= grandstream-snmp-exporter:latest
BIN ?= grandstream-snmp-exporter
PKG ?= ./cmd/exporter

.PHONY: tidy build test image run-ap run-switch run-router run-gcc k8s-servicemonitor

tidy:
	go mod tidy

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN) $(PKG)

test:
	go test ./...

image:
	podman build -t $(IMAGE) .

run-ap:
	podman run --rm -p 9109:9109 \
	  -e DEVICE_TYPE=AP \
	  -e DEVICE_IP=192.0.2.10 \
	  -e SNMP_VERSION=2c \
	  -e SNMP_COMMUNITY=public \
	  $(IMAGE)

run-switch:
	podman run --rm -p 9109:9109 \
	  -e DEVICE_TYPE=SWITCH \
	  -e DEVICE_IP=192.0.2.12 \
	  -e SNMP_VERSION=2c \
	  -e SNMP_COMMUNITY=public \
	  $(IMAGE)

run-router:
	podman run --rm -p 9109:9109 \
	  -e DEVICE_TYPE=ROUTER \
	  -e DEVICE_IP=192.0.2.11 \
	  -e SNMP_VERSION=3 \
	  -e SNMP_SECURITY_LEVEL=authPriv \
	  -e SNMP_USERNAME=myuser \
	  -e SNMP_AUTH_PROTOCOL=SHA-256 \
	  -e SNMP_AUTH_PASSPHRASE=auth-pass \
	  -e SNMP_PRIV_PROTOCOL=AES \
	  -e SNMP_PRIV_PASSPHRASE=priv-pass \
	  $(IMAGE)

run-gcc:
	podman run --rm -p 9109:9109 \
	  -e DEVICE_TYPE=GCC \
	  -e DEVICE_IP=192.0.2.20 \
	  -e SNMP_VERSION=3 \
	  -e SNMP_SECURITY_LEVEL=authNoPriv \
	  -e SNMP_USERNAME=myuser \
	  -e SNMP_AUTH_PROTOCOL=SHA-256 \
	  -e SNMP_AUTH_PASSPHRASE=auth-pass \
	  $(IMAGE)

k8s-servicemonitor:
	@cat k8s/servicemonitor.yaml
