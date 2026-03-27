# Makefile Usage Guide
## Grandstream SNMP Exporter

This document explains how to use the included `Makefile` to simplify building,
testing, running, and deploying the Grandstream SNMP Exporter.

The Makefile acts as a shortcut layer so you do not need to remember long
Go build commands or container commands.

---

# Why Use the Makefile?

Instead of typing long commands like:

    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o grandstream-snmp-exporter ./cmd/exporter

You can simply run:

    make build

This improves:
- Developer productivity
- Standardization across environments
- CI/CD automation
- Team onboarding

---

# Available Targets

## make tidy

Cleans and synchronizes Go module dependencies.

    make tidy

Runs:

    go mod tidy

Use when:
- You add new imports
- You remove packages
- Dependencies change

---

## make build

Builds the local Go binary.

    make build

Output:
    ./grandstream-snmp-exporter

Use when:
- Testing locally without containers
- Debugging
- Running exporter directly on a server

Example local run:

    DEVICE_TYPE=AP \
    DEVICE_IP=192.0.2.10 \
    SNMP_VERSION=2c \
    SNMP_COMMUNITY=public \
    ./grandstream-snmp-exporter

---

## make test

Runs all Go tests.

    make test

Runs:

    go test ./...

Use when:
- Validating changes
- Running CI pipelines

---

## make image

Builds the container image using Podman.

    make image

Runs:

    podman build -t grandstream-snmp-exporter:latest .

Use when:
- Preparing container for Kubernetes
- Preparing image for registry push

---

## make run-ap

Runs the container with example configuration for an AP device (SNMP v2c).

    make run-ap

You can edit the IP and credentials inside the Makefile to match your lab.

---

## make run-switch

Runs the container with example configuration for a Switch device.

    make run-switch

---

## make run-router

Runs the container with SNMP v3 authPriv example for Router.

    make run-router

---

## make run-gcc

Runs the container with SNMP v3 authNoPriv example for GCC.

    make run-gcc

---

## make k8s-servicemonitor

Prints the ServiceMonitor YAML to stdout.

    make k8s-servicemonitor

Example usage:

    make k8s-servicemonitor | kubectl apply -f -

---

# Typical Development Workflow

## Local Development

    make tidy
    make build
    ./grandstream-snmp-exporter

## Container Build + Test

    make image
    make run-switch

## Kubernetes Deployment

    kubectl apply -f k8s/
    make k8s-servicemonitor | kubectl apply -f -

---

# Customizing the Makefile

You can modify the following variables at the top of the Makefile:

    IMAGE ?= grandstream-snmp-exporter:latest
    BIN ?= grandstream-snmp-exporter

Examples:

Build with custom tag:

    make image IMAGE=myrepo/grandstream-exporter:v1.0.0

---

# Advanced Extensions (Optional)

You can extend the Makefile with:

- make push        (push image to registry)
- make release     (version tagging)
- make multiarch   (amd64 + arm64 builds)
- make lint        (golangci-lint)
- make helm-deploy (Helm installation)

---

# Summary

The Makefile provides:

- Fast local development
- Standardized build process
- Container image automation
- Easier Kubernetes integration

It is strongly recommended to use `make` targets instead of manually typing build commands,
especially in team or CI/CD environments.
