package exporter

import (
	"testing"

	"github.com/gosnmp/gosnmp"
)

func TestFindSymbolRequiresOIDBoundary(t *testing.T) {
	hint, idx, ok := findSymbol(".1.3.6.1.2.1.31.1.1.1.10.42")
	if !ok || hint.Symbol != "ifHCOutOctets" || idx != "42" {
		t.Fatalf("unexpected lookup: hint=%+v idx=%q ok=%v", hint, idx, ok)
	}

	if _, _, ok := findSymbol(".1.3.6.1.2.1.31.1.1.1.100.42"); ok {
		t.Fatal("prefix without an OID component boundary must not match")
	}
}

func TestPDUToFloat64(t *testing.T) {
	tests := []struct {
		name string
		pdu  gosnmp.SnmpPDU
		want float64
	}{
		{name: "signed", pdu: gosnmp.SnmpPDU{Value: int32(-42)}, want: -42},
		{name: "counter64", pdu: gosnmp.SnmpPDU{Value: uint64(1 << 40)}, want: 1 << 40},
		{name: "timeticks", pdu: gosnmp.SnmpPDU{Type: gosnmp.TimeTicks, Value: uint32(12345)}, want: 123.45},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := pduToFloat64(test.pdu)
			if !ok || got != test.want {
				t.Fatalf("pduToFloat64() = %v, %v; want %v, true", got, ok, test.want)
			}
		})
	}
}

func TestPDUToStringTrimsFirmwareNUL(t *testing.T) {
	pdu := gosnmp.SnmpPDU{Value: []byte("1.0.25.18\x00")}
	if got := pduToString(pdu); got != "1.0.25.18" {
		t.Fatalf("pduToString() = %q", got)
	}
}
