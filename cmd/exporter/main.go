package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"example.com/grandstream-snmp-exporter/internal/exporter"
	"example.com/grandstream-snmp-exporter/internal/snmp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var %s", key)
	}
	return v
}

func main() {
	deviceType := mustEnv("DEVICE_TYPE")
	deviceIP := mustEnv("DEVICE_IP")
	snmpVer := mustEnv("SNMP_VERSION")

	port := uint16(161)
	if p := os.Getenv("SNMP_PORT"); p != "" {
		pi, err := strconv.Atoi(p)
		if err != nil || pi < 1 || pi > 65535 {
			log.Fatalf("invalid SNMP_PORT: %q", p)
		}
		port = uint16(pi)
	}

	cfg := snmp.Config{
		Target:  deviceIP,
		Port:    port,
		Timeout: 8 * time.Second,
		Retries: 1,
		Version: snmp.Version(snmpVer),
	}

	switch snmpVer {
	case "2c":
		cfg.Community = mustEnv("SNMP_COMMUNITY")
	case "3":
		cfg.SecLevel = snmp.V3SecLevel(mustEnv("SNMP_SECURITY_LEVEL"))
		cfg.Username = mustEnv("SNMP_USERNAME")
		cfg.AuthProto = os.Getenv("SNMP_AUTH_PROTOCOL")
		cfg.AuthPass = os.Getenv("SNMP_AUTH_PASSPHRASE")
		cfg.PrivProto = os.Getenv("SNMP_PRIV_PROTOCOL")
		cfg.PrivPass = os.Getenv("SNMP_PRIV_PASSPHRASE")
	default:
		log.Fatalf("unsupported SNMP_VERSION %q, use 2c or 3", snmpVer)
	}

	client, err := snmp.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(&exporter.Collector{
		SNMP:       client,
		DeviceType: deviceType,
		DeviceIP:   deviceIP,
	})

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	listen := os.Getenv("LISTEN")
	if listen == "" {
		listen = ":9109"
	}

	log.Printf("listening on %s", listen)
	log.Fatal(http.ListenAndServe(listen, mux))
}