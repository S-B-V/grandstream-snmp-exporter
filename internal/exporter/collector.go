package exporter

import (
    "encoding/hex"
    "fmt"
    "log"
    "regexp"
    "strings"
    "unicode/utf8"

    "github.com/S-B-V/grandstream-snmp-exporter/internal/snmp"
    "github.com/gosnmp/gosnmp"
    "github.com/prometheus/client_golang/prometheus"
)

var promNameRe = regexp.MustCompile(`[^a-zA-Z0-9_]`)

type Collector struct {
    Config     snmp.Config
    DeviceType string
    DeviceIP   string
}

type symbolHint struct {
    Module      string
    Symbol      string
    ValueType   string // gauge|string
    IndexName   string
    Description string
}

var exactSymbolMap = map[string]symbolHint{
    ".1.3.6.1.2.1.1.1.0":  {Module: "SNMPv2-MIB", Symbol: "sysDescr", ValueType: "string", Description: "System description"},
    ".1.3.6.1.2.1.25.1.1.0": {Module: "HOST-RESOURCES-MIB", Symbol: "hrSystemUptime", ValueType: "gauge", Description: "System uptime in seconds"},
    ".1.3.6.1.2.1.25.1.6.0": {Module: "HOST-RESOURCES-MIB", Symbol: "hrSystemProcesses", ValueType: "gauge", Description: "Running processes"},
    ".1.3.6.1.2.1.25.2.2.0": {Module: "HOST-RESOURCES-MIB", Symbol: "hrMemorySize", ValueType: "gauge", Description: "Main memory size in KBytes"},
}

var prefixSymbolMap = map[string]symbolHint{
    // Storage
    ".1.3.6.1.2.1.25.2.3.1.3": {Module: "HOST-RESOURCES-MIB", Symbol: "hrStorageDescr", ValueType: "string", IndexName: "hrStorageIndex"},
    ".1.3.6.1.2.1.25.2.3.1.4": {Module: "HOST-RESOURCES-MIB", Symbol: "hrStorageAllocationUnits", ValueType: "gauge", IndexName: "hrStorageIndex"},
    ".1.3.6.1.2.1.25.2.3.1.5": {Module: "HOST-RESOURCES-MIB", Symbol: "hrStorageSize", ValueType: "gauge", IndexName: "hrStorageIndex"},
    ".1.3.6.1.2.1.25.2.3.1.6": {Module: "HOST-RESOURCES-MIB", Symbol: "hrStorageUsed", ValueType: "gauge", IndexName: "hrStorageIndex"},
    // CPU
    ".1.3.6.1.2.1.25.3.3.1.2": {Module: "HOST-RESOURCES-MIB", Symbol: "hrProcessorLoad", ValueType: "gauge", IndexName: "hrProcessorIndex"},
    // Interfaces (IF-MIB)
    ".1.3.6.1.2.1.2.2.1.2":  {Module: "IF-MIB", Symbol: "ifDescr", ValueType: "string", IndexName: "ifIndex"},
    ".1.3.6.1.2.1.2.2.1.10": {Module: "IF-MIB", Symbol: "ifInOctets", ValueType: "gauge", IndexName: "ifIndex"},
    ".1.3.6.1.2.1.2.2.1.11": {Module: "IF-MIB", Symbol: "ifInUcastPkts", ValueType: "gauge", IndexName: "ifIndex"},
    ".1.3.6.1.2.1.2.2.1.12": {Module: "IF-MIB", Symbol: "ifInNUcastPkts", ValueType: "gauge", IndexName: "ifIndex"},
    ".1.3.6.1.2.1.2.2.1.13": {Module: "IF-MIB", Symbol: "ifInDiscards", ValueType: "gauge", IndexName: "ifIndex"},
    ".1.3.6.1.2.1.2.2.1.14": {Module: "IF-MIB", Symbol: "ifInErrors", ValueType: "gauge", IndexName: "ifIndex"},
    ".1.3.6.1.2.1.2.2.1.16": {Module: "IF-MIB", Symbol: "ifOutOctets", ValueType: "gauge", IndexName: "ifIndex"},
    ".1.3.6.1.2.1.2.2.1.17": {Module: "IF-MIB", Symbol: "ifOutUcastPkts", ValueType: "gauge", IndexName: "ifIndex"},
    ".1.3.6.1.2.1.2.2.1.18": {Module: "IF-MIB", Symbol: "ifOutNUcastPkts", ValueType: "gauge", IndexName: "ifIndex"},
    ".1.3.6.1.2.1.2.2.1.19": {Module: "IF-MIB", Symbol: "ifOutDiscards", ValueType: "gauge", IndexName: "ifIndex"},
    ".1.3.6.1.2.1.2.2.1.20": {Module: "IF-MIB", Symbol: "ifOutErrors", ValueType: "gauge", IndexName: "ifIndex"},
    ".1.3.6.1.2.1.2.2.1.21": {Module: "IF-MIB", Symbol: "ifOutQLen", ValueType: "gauge", IndexName: "ifIndex"},
}

func sanitizeMetricName(s string) string {
    return strings.ToLower(promNameRe.ReplaceAllString(s, "_"))
}

func pduToFloat64(pdu gosnmp.SnmpPDU) (float64, bool) {
    var val float64
    var ok bool

    switch v := pdu.Value.(type) {
    case int, int32, int64, uint, uint32, uint64:
        val, ok = float64(gosnmp.ToBigInt(v).Uint64()), true
    case []byte:
        if pdu.Type == gosnmp.TimeTicks && len(v) > 0 {
            var n uint64
            for _, b := range v {
                n = (n << 8) | uint64(b)
            }
            val, ok = float64(n), true
        }
    }

    if ok && pdu.Type == gosnmp.TimeTicks {
        return val / 100.0, true
    }

    return val, ok
}

func pduToString(pdu gosnmp.SnmpPDU) string {
    switch v := pdu.Value.(type) {
    case string:
        return v
    case []byte:
        if utf8.Valid(v) {
            return string(v)
        }
        return hex.EncodeToString(v)
    default:
        return fmt.Sprintf("%v", pdu.Value)
    }
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
    ch <- prometheus.NewDesc("grandstream_exporter_up", "SNMP status", []string{"device_ip"}, nil)
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
    client, err := snmp.New(c.Config)
    if err != nil {
        log.Printf("failed to initialize SNMP client for %s: %v", c.DeviceIP, err)
        ch <- prometheus.MustNewConstMetric(
            prometheus.NewDesc("grandstream_exporter_up", "SNMP status", []string{"device_ip"}, nil),
            prometheus.GaugeValue,
            0,
            c.DeviceIP,
        )
        return
    }

    if err := client.Connect(); err != nil {
        log.Printf("failed to connect to %s via SNMP: %v", c.DeviceIP, err)
        ch <- prometheus.MustNewConstMetric(
            prometheus.NewDesc("grandstream_exporter_up", "SNMP status", []string{"device_ip"}, nil),
            prometheus.GaugeValue,
            0,
            c.DeviceIP,
        )
        return
    }
    defer client.Close()

    ch <- prometheus.MustNewConstMetric(
        prometheus.NewDesc("grandstream_exporter_up", "SNMP status", []string{"device_ip"}, nil),
        prometheus.GaugeValue,
        1,
        c.DeviceIP,
    )

    type storageData struct {
        Units float64
        Size  float64
        Used  float64
        Descr string
    }

    storageMap := make(map[string]*storageData)

    process := func(pdu gosnmp.SnmpPDU) {
        hint, idx, ok := func(oid string) (symbolHint, string, bool) {
            if h, found := exactSymbolMap[oid]; found {
                return h, "", true
            }
            for prefix, h := range prefixSymbolMap {
                if strings.HasPrefix(oid, prefix) {
                    return h, strings.TrimPrefix(strings.TrimPrefix(oid, prefix), "."), true
                }
            }
            return symbolHint{}, "", false
        }(pdu.Name)
        if !ok {
            return
        }

        if hint.Module == "HOST-RESOURCES-MIB" && hint.IndexName == "hrStorageIndex" {
            if _, exists := storageMap[idx]; !exists {
                storageMap[idx] = &storageData{}
            }
            val, _ := pduToFloat64(pdu)
            switch hint.Symbol {
            case "hrStorageAllocationUnits":
                storageMap[idx].Units = val
            case "hrStorageSize":
                storageMap[idx].Size = val
            case "hrStorageUsed":
                storageMap[idx].Used = val
            case "hrStorageDescr":
                storageMap[idx].Descr = pduToString(pdu)
            }
        }

        metricName := "grandstream_" + sanitizeMetricName(hint.Module+"_"+hint.Symbol)
        labelNames := []string{"device_ip", "device_type"}
        labelValues := []string{c.DeviceIP, c.DeviceType}

        if idx != "" {
            labelNames = append(labelNames, hint.IndexName)
            labelValues = append(labelValues, idx)
        }

        if hint.ValueType == "gauge" {
            if f, ok := pduToFloat64(pdu); ok {
                ch <- prometheus.MustNewConstMetric(
                    prometheus.NewDesc(metricName, "", labelNames, nil),
                    prometheus.GaugeValue,
                    f,
                    labelValues...,
                )
            }
            return
        }

        labelNames = append(labelNames, "value")
        labelValues = append(labelValues, pduToString(pdu))
        ch <- prometheus.MustNewConstMetric(
            prometheus.NewDesc(metricName+"_info", "", labelNames, nil),
            prometheus.GaugeValue,
            1,
            labelValues...,
        )
    }

    scalarOIDs := make([]string, 0, len(exactSymbolMap))
    for oid := range exactSymbolMap {
        scalarOIDs = append(scalarOIDs, oid)
    }

    if res, err := client.Get(scalarOIDs); err == nil {
        for _, pdu := range res.Variables {
            process(pdu)
        }
    } else {
        log.Printf("SNMP GET failed for %s: %v", c.DeviceIP, err)
    }

    tableRoots := []string{
        ".1.3.6.1.2.1.25.2.3.1",
        ".1.3.6.1.2.1.25.3.3.1",
        ".1.3.6.1.2.1.2.2.1",
    }
    for _, root := range tableRoots {
        if err := client.BulkWalk(root, func(pdu gosnmp.SnmpPDU) error {
            process(pdu)
            return nil
        }); err != nil {
            log.Printf("SNMP BULKWALK failed for %s on %s: %v", c.DeviceIP, root, err)
        }
    }

    for idx, s := range storageMap {
        if s.Units <= 0 {
            continue
        }
        div := 1024.0 * 1024.0
        labels := []string{"device_ip", "device_type", "index", "descr"}
        values := []string{c.DeviceIP, c.DeviceType, idx, s.Descr}

        ch <- prometheus.MustNewConstMetric(
            prometheus.NewDesc("grandstream_storage_total_megabytes", "", labels, nil),
            prometheus.GaugeValue,
            (s.Size*s.Units)/div,
            values...,
        )
        ch <- prometheus.MustNewConstMetric(
            prometheus.NewDesc("grandstream_storage_used_megabytes", "", labels, nil),
            prometheus.GaugeValue,
            (s.Used*s.Units)/div,
            values...,
        )
    }
}
