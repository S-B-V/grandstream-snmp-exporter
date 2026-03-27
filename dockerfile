FROM --platform=linux/amd64 golang:1.23-alpine AS build

WORKDIR /src

COPY . .

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -trimpath -ldflags="-s -w" -o /exporter ./cmd/exporter

FROM scratch

LABEL org.opencontainers.image.source="https://github.com/S-B-V/grandstream-snmp-exporter"

WORKDIR /

COPY --from=build /exporter /exporter
COPY mibs/standard /mibs/standard

ENV MIB_STANDARD_DIR=/mibs/standard

EXPOSE 9109

ENTRYPOINT ["/exporter"]