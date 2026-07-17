package exporter

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gosnmp/gosnmp"
	"github.com/prometheus/client_golang/prometheus"
)

var promNameRe = regexp.MustCompile(`[^a-zA-Z0-9_]`)

type Collector struct {
	SNMP       *gosnmp.GoSNMP
	DeviceType string
	DeviceIP   string
}

type symbolHint struct {
	Module      string
	Symbol      string
	ValueType   string // gauge|counter|string
	IndexName   string
	Description string
}

var exactSymbolMap = map[string]symbolHint{
	".1.3.6.1.2.1.1.1.0":    {Module: "SNMPv2-MIB", Symbol: "sysDescr", ValueType: "string", Description: "System description"},
	".1.3.6.1.2.1.1.5.0":    {Module: "SNMPv2-MIB", Symbol: "sysName", ValueType: "string", Description: "System name"},
	".1.3.6.1.2.1.25.1.1.0": {Module: "HOST-RESOURCES-MIB", Symbol: "hrSystemUptime", ValueType: "gauge", Description: "System uptime in seconds"},
	".1.3.6.1.2.1.25.1.6.0": {Module: "HOST-RESOURCES-MIB", Symbol: "hrSystemProcesses", ValueType: "gauge", Description: "Running processes"},
	".1.3.6.1.2.1.25.2.2.0": {Module: "HOST-RESOURCES-MIB", Symbol: "hrMemorySize", ValueType: "gauge", Description: "Main memory size in KBytes"},
}

var prefixSymbolMap = map[string]symbolHint{
	// Storage
	".1.3.6.1.2.1.25.2.3.1.2": {Module: "HOST-RESOURCES-MIB", Symbol: "hrStorageType", ValueType: "string", IndexName: "hrStorageIndex"},
	".1.3.6.1.2.1.25.2.3.1.3": {Module: "HOST-RESOURCES-MIB", Symbol: "hrStorageDescr", ValueType: "string", IndexName: "hrStorageIndex"},
	".1.3.6.1.2.1.25.2.3.1.4": {Module: "HOST-RESOURCES-MIB", Symbol: "hrStorageAllocationUnits", ValueType: "gauge", IndexName: "hrStorageIndex"},
	".1.3.6.1.2.1.25.2.3.1.5": {Module: "HOST-RESOURCES-MIB", Symbol: "hrStorageSize", ValueType: "gauge", IndexName: "hrStorageIndex"},
	".1.3.6.1.2.1.25.2.3.1.6": {Module: "HOST-RESOURCES-MIB", Symbol: "hrStorageUsed", ValueType: "gauge", IndexName: "hrStorageIndex"},
	// CPU
	".1.3.6.1.2.1.25.3.3.1.2": {Module: "HOST-RESOURCES-MIB", Symbol: "hrProcessorLoad", ValueType: "gauge", IndexName: "hrProcessorIndex"},
	// Interfaces (IF-MIB)
	".1.3.6.1.2.1.2.2.1.2":  {Module: "IF-MIB", Symbol: "ifDescr", ValueType: "string", IndexName: "ifIndex"},
	".1.3.6.1.2.1.2.2.1.5":  {Module: "IF-MIB", Symbol: "ifSpeed", ValueType: "gauge", IndexName: "ifIndex"},
	".1.3.6.1.2.1.2.2.1.7":  {Module: "IF-MIB", Symbol: "ifAdminStatus", ValueType: "gauge", IndexName: "ifIndex"},
	".1.3.6.1.2.1.2.2.1.8":  {Module: "IF-MIB", Symbol: "ifOperStatus", ValueType: "gauge", IndexName: "ifIndex"},
	".1.3.6.1.2.1.2.2.1.10": {Module: "IF-MIB", Symbol: "ifInOctets", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.2.2.1.11": {Module: "IF-MIB", Symbol: "ifInUcastPkts", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.2.2.1.12": {Module: "IF-MIB", Symbol: "ifInNUcastPkts", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.2.2.1.13": {Module: "IF-MIB", Symbol: "ifInDiscards", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.2.2.1.14": {Module: "IF-MIB", Symbol: "ifInErrors", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.2.2.1.16": {Module: "IF-MIB", Symbol: "ifOutOctets", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.2.2.1.17": {Module: "IF-MIB", Symbol: "ifOutUcastPkts", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.2.2.1.18": {Module: "IF-MIB", Symbol: "ifOutNUcastPkts", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.2.2.1.19": {Module: "IF-MIB", Symbol: "ifOutDiscards", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.2.2.1.20": {Module: "IF-MIB", Symbol: "ifOutErrors", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.2.2.1.21": {Module: "IF-MIB", Symbol: "ifOutQLen", ValueType: "gauge", IndexName: "ifIndex"},
	// 64-bit interface counters from ifXTable. These avoid the frequent wraps of
	// the legacy 32-bit counters on busy links.
	".1.3.6.1.2.1.31.1.1.1.1":  {Module: "IF-MIB", Symbol: "ifName", ValueType: "string", IndexName: "ifIndex"},
	".1.3.6.1.2.1.31.1.1.1.6":  {Module: "IF-MIB", Symbol: "ifHCInOctets", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.31.1.1.1.7":  {Module: "IF-MIB", Symbol: "ifHCInUcastPkts", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.31.1.1.1.8":  {Module: "IF-MIB", Symbol: "ifHCInMulticastPkts", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.31.1.1.1.9":  {Module: "IF-MIB", Symbol: "ifHCInBroadcastPkts", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.31.1.1.1.10": {Module: "IF-MIB", Symbol: "ifHCOutOctets", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.31.1.1.1.11": {Module: "IF-MIB", Symbol: "ifHCOutUcastPkts", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.31.1.1.1.12": {Module: "IF-MIB", Symbol: "ifHCOutMulticastPkts", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.31.1.1.1.13": {Module: "IF-MIB", Symbol: "ifHCOutBroadcastPkts", ValueType: "counter", IndexName: "ifIndex"},
	".1.3.6.1.2.1.31.1.1.1.15": {Module: "IF-MIB", Symbol: "ifHighSpeed", ValueType: "gauge", IndexName: "ifIndex"},
	".1.3.6.1.2.1.31.1.1.1.18": {Module: "IF-MIB", Symbol: "ifAlias", ValueType: "string", IndexName: "ifIndex"},
}

func sanitizeMetricName(s string) string {
	return strings.ToLower(promNameRe.ReplaceAllString(s, "_"))
}

func pduToFloat64(pdu gosnmp.SnmpPDU) (float64, bool) {
	var val float64
	var ok bool
	switch v := pdu.Value.(type) {
	case int:
		val, ok = float64(v), true
	case int32:
		val, ok = float64(v), true
	case int64:
		val, ok = float64(v), true
	case uint:
		val, ok = float64(v), true
	case uint32:
		val, ok = float64(v), true
	case uint64:
		val, ok = float64(v), true
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
		return strings.TrimRight(v, "\x00")
	case []byte:
		if utf8.Valid(v) {
			return strings.TrimRight(string(v), "\x00")
		}
		return hex.EncodeToString(v)
	}
	return fmt.Sprintf("%v", pdu.Value)
}

func findSymbol(oid string) (symbolHint, string, bool) {
	if h, ok := exactSymbolMap[oid]; ok {
		return h, "", true
	}
	for prefix, h := range prefixSymbolMap {
		if strings.HasPrefix(oid, prefix+".") {
			return h, strings.TrimPrefix(oid, prefix+"."), true
		}
	}
	return symbolHint{}, "", false
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("grandstream_exporter_up", "SNMP status", []string{"device_ip"}, nil)
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	if err := c.SNMP.Connect(); err != nil {
		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("grandstream_exporter_up", "", []string{"device_ip"}, nil), prometheus.GaugeValue, 0, c.DeviceIP)
		return
	}
	defer c.SNMP.Conn.Close()
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("grandstream_exporter_up", "", []string{"device_ip"}, nil), prometheus.GaugeValue, 1, c.DeviceIP)

	type storageData struct {
		Units, Size, Used float64
		Descr, Type       string
	}
	storageMap := make(map[string]*storageData)
	ifNames := make(map[string]string)
	processorCores := make(map[string]string)
	nextCore := 0

	// Fetch interface names first so every interface sample can carry a useful
	// label instead of forcing dashboards to display opaque numeric indices.
	_ = c.SNMP.BulkWalk(".1.3.6.1.2.1.31.1.1.1.1", func(p gosnmp.SnmpPDU) error {
		idx := strings.TrimPrefix(p.Name, ".1.3.6.1.2.1.31.1.1.1.1.")
		ifNames[idx] = pduToString(p)
		return nil
	})
	_ = c.SNMP.BulkWalk(".1.3.6.1.2.1.2.2.1.2", func(p gosnmp.SnmpPDU) error {
		idx := strings.TrimPrefix(p.Name, ".1.3.6.1.2.1.2.2.1.2.")
		if _, exists := ifNames[idx]; !exists {
			ifNames[idx] = pduToString(p)
		}
		return nil
	})

	process := func(pdu gosnmp.SnmpPDU) {
		hint, idx, ok := findSymbol(pdu.Name)
		if !ok {
			return
		}

		if hint.Module == "HOST-RESOURCES-MIB" && hint.IndexName == "hrStorageIndex" {
			if _, exists := storageMap[idx]; !exists {
				storageMap[idx] = &storageData{}
			}
			val, _ := pduToFloat64(pdu)
			switch hint.Symbol {
			case "hrStorageType":
				storageMap[idx].Type = pduToString(pdu)
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

		mName := "grandstream_" + sanitizeMetricName(hint.Module+"_"+hint.Symbol)
		lbls, vals := []string{"device_ip", "device_type"}, []string{c.DeviceIP, c.DeviceType}
		if idx != "" {
			lbls, vals = append(lbls, hint.IndexName), append(vals, idx)
		}
		if hint.Module == "IF-MIB" && idx != "" {
			lbls, vals = append(lbls, "if_name"), append(vals, ifNames[idx])
		}
		if hint.Symbol == "hrProcessorLoad" {
			core, exists := processorCores[idx]
			if !exists {
				core = fmt.Sprintf("%d", nextCore)
				processorCores[idx] = core
				nextCore++
			}
			lbls, vals = append(lbls, "core"), append(vals, core)
		}

		if hint.ValueType == "gauge" || hint.ValueType == "counter" {
			if f, ok := pduToFloat64(pdu); ok {
				valueType := prometheus.GaugeValue
				if hint.ValueType == "counter" {
					valueType = prometheus.CounterValue
				}
				ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(mName, hint.Description, lbls, nil), valueType, f, vals...)
			}
		} else {
			lbls, vals = append(lbls, "value"), append(vals, pduToString(pdu))
			ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(mName+"_info", "", lbls, nil), prometheus.GaugeValue, 1, vals...)
		}
	}

	var scalarOIDs []string
	for k := range exactSymbolMap {
		scalarOIDs = append(scalarOIDs, k)
	}
	if res, err := c.SNMP.Get(scalarOIDs); err == nil {
		for _, p := range res.Variables {
			process(p)
		}
	}

	tableRoots := []string{".1.3.6.1.2.1.25.2.3.1", ".1.3.6.1.2.1.25.3.3.1", ".1.3.6.1.2.1.2.2.1", ".1.3.6.1.2.1.31.1.1.1"}
	for _, t := range tableRoots {
		_ = c.SNMP.BulkWalk(t, func(p gosnmp.SnmpPDU) error { process(p); return nil })
	}

	for idx, s := range storageMap {
		if s.Units > 0 {
			div := 1024.0 * 1024.0
			l, v := []string{"device_ip", "device_type", "index", "descr", "storage_type"}, []string{c.DeviceIP, c.DeviceType, idx, s.Descr, s.Type}
			ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("grandstream_storage_total_megabytes", "", l, nil), prometheus.GaugeValue, (s.Size*s.Units)/div, v...)
			ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("grandstream_storage_used_megabytes", "", l, nil), prometheus.GaugeValue, (s.Used*s.Units)/div, v...)
			if s.Size > 0 {
				ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("grandstream_storage_utilization_ratio", "Storage utilization from 0 to 1", l, nil), prometheus.GaugeValue, s.Used/s.Size, v...)
			}
		}
	}

	if strings.EqualFold(c.DeviceType, "AP") {
		c.collectAP(ch)
	}
}

func (c *Collector) collectAP(ch chan<- prometheus.Metric) {
	const systemBase = ".1.3.6.1.4.1.42397.1.1.2"
	scalarNames := []string{"model", "name", "mac", "firmware", "reported_ip"}
	oids := make([]string, len(scalarNames))
	for i := range scalarNames {
		oids[i] = fmt.Sprintf("%s.%d.0", systemBase, i+1)
	}
	if result, err := c.SNMP.Get(oids); err == nil {
		values := map[string]string{}
		for i, pdu := range result.Variables {
			if i < len(scalarNames) {
				values[scalarNames[i]] = pduToString(pdu)
			}
		}
		labels := []string{"device_ip", "device_type", "model", "name", "mac", "firmware", "reported_ip"}
		labelValues := []string{c.DeviceIP, c.DeviceType, values["model"], values["name"], values["mac"], values["firmware"], values["reported_ip"]}
		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("grandstream_ap_device_info", "Grandstream AP identity and firmware", labels, nil), prometheus.GaugeValue, 1, labelValues...)
	}

	type radioData struct {
		name                                                     string
		status, channel, power, txBytes, rxBytes, txDrops, rxBad float64
	}
	radios := map[string]*radioData{}
	const radioBase = ".1.3.6.1.4.1.42397.1.1.3.1.1"
	_ = c.SNMP.BulkWalk(radioBase, func(pdu gosnmp.SnmpPDU) error {
		rest := strings.TrimPrefix(pdu.Name, radioBase+".")
		parts := strings.SplitN(rest, ".", 2)
		if len(parts) != 2 {
			return nil
		}
		column, idx := parts[0], parts[1]
		if _, exists := radios[idx]; !exists {
			radios[idx] = &radioData{}
		}
		radio := radios[idx]
		value, _ := pduToFloat64(pdu)
		switch column {
		case "2":
			radio.name = pduToString(pdu)
		case "3":
			radio.status = value
		case "4":
			radio.channel = value
		case "5":
			radio.power = value
		case "9":
			radio.txBytes = value
		case "10":
			radio.txDrops = value
		case "13":
			radio.rxBytes = value
		case "15":
			radio.rxBad = value
		}
		return nil
	})
	for idx, radio := range radios {
		labels := []string{"device_ip", "device_type", "radio", "radio_name"}
		values := []string{c.DeviceIP, c.DeviceType, idx, radio.name}
		emit := func(name, help string, value float64, valueType prometheus.ValueType) {
			ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(name, help, labels, nil), valueType, value, values...)
		}
		emit("grandstream_ap_radio_status", "Radio status reported by the AP", radio.status, prometheus.GaugeValue)
		emit("grandstream_ap_radio_channel", "Configured radio channel", radio.channel, prometheus.GaugeValue)
		emit("grandstream_ap_radio_transmit_power", "Configured radio transmit power", radio.power, prometheus.GaugeValue)
		emit("grandstream_ap_radio_tx_bytes", "Bytes transmitted by the radio", radio.txBytes, prometheus.CounterValue)
		emit("grandstream_ap_radio_rx_bytes", "Bytes received by the radio", radio.rxBytes, prometheus.CounterValue)
		emit("grandstream_ap_radio_tx_drops", "Transmit drops reported by the radio", radio.txDrops, prometheus.CounterValue)
		emit("grandstream_ap_radio_rx_bad", "Bad receive frames reported by the radio", radio.rxBad, prometheus.CounterValue)
	}

	type clientData struct {
		mac, ip, wlanMac, ssid, hostname, os string
		signal, assocSeconds                 float64
	}
	clients := map[string]*clientData{}
	const clientBase = ".1.3.6.1.4.1.42397.1.1.3.3.1"
	_ = c.SNMP.BulkWalk(clientBase, func(pdu gosnmp.SnmpPDU) error {
		rest := strings.TrimPrefix(pdu.Name, clientBase+".")
		parts := strings.SplitN(rest, ".", 2)
		if len(parts) != 2 {
			return nil
		}
		column, key := parts[0], parts[1]
		if _, exists := clients[key]; !exists {
			clients[key] = &clientData{}
		}
		client := clients[key]
		value, _ := pduToFloat64(pdu)
		switch column {
		case "1":
			client.mac = pduToString(pdu)
		case "2":
			client.ip = pduToString(pdu)
		case "3":
			client.wlanMac = pduToString(pdu)
		case "4":
			client.ssid = pduToString(pdu)
		case "5":
			client.signal = value
		case "6":
			client.assocSeconds = value
		case "8":
			client.hostname = pduToString(pdu)
		case "9":
			client.os = pduToString(pdu)
		}
		return nil
	})
	baseLabels := []string{"device_ip", "device_type"}
	baseValues := []string{c.DeviceIP, c.DeviceType}
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("grandstream_ap_clients", "Currently connected wireless clients", baseLabels, nil), prometheus.GaugeValue, float64(len(clients)), baseValues...)
	for _, client := range clients {
		labels := []string{"device_ip", "device_type", "client_mac", "client_ip", "wlan_mac", "ssid", "hostname", "os"}
		values := []string{c.DeviceIP, c.DeviceType, client.mac, client.ip, client.wlanMac, client.ssid, client.hostname, client.os}
		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("grandstream_ap_client_info", "Currently connected wireless client", labels, nil), prometheus.GaugeValue, 1, values...)
		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("grandstream_ap_client_signal", "Wireless client signal value reported by the AP", labels, nil), prometheus.GaugeValue, client.signal, values...)
		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("grandstream_ap_client_association_seconds", "Wireless client association duration", labels, nil), prometheus.GaugeValue, client.assocSeconds, values...)
	}
}
