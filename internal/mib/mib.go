package mib

import (
	"os"
	"strings"
	"sync"

	"github.com/sleepinggenius2/gosmi"
)

var (
	once       sync.Once
	loaderInit bool
)

func Init(_ string) error {
	once.Do(func() {
		standardDir := os.Getenv("MIB_STANDARD_DIR")
		if standardDir == "" {
			standardDir = "/mibs/standard"
		}

		gosmi.Init()
		loaderInit = true

		if st, err := os.Stat(standardDir); err == nil && st.IsDir() {
			gosmi.AppendPath(standardDir)
		}
	})

	return nil
}

type Resolved struct {
	Module         string
	Symbol         string
	BaseOID        string
	InstanceSuffix string
}

func ResolveOID(oid string) (*Resolved, bool) {
	if !loaderInit {
		return nil, false
	}

	n, err := gosmi.GetNode(oid)
	if err == nil && n.Name != "" {
		base := n.Oid.String()
		return &Resolved{
			Module:         "",
			Symbol:         n.Name,
			BaseOID:        base,
			InstanceSuffix: strings.TrimPrefix(oid, base),
		}, true
	}

	trimmed := strings.TrimPrefix(oid, ".")
	parts := strings.Split(trimmed, ".")
	for i := len(parts) - 1; i > 0; i-- {
		tryOID := "." + strings.Join(parts[:i], ".")
		n2, err2 := gosmi.GetNode(tryOID)
		if err2 == nil && n2.Name != "" {
			base := n2.Oid.String()
			return &Resolved{
				Module:         "",
				Symbol:         n2.Name,
				BaseOID:        base,
				InstanceSuffix: strings.TrimPrefix(oid, base),
			}, true
		}
	}

	return nil, false
}

func IsNumeric(_ string) bool {
	return false
}

type IndexDef struct {
	Name string
}

func IndexesForOID(_ string) ([]IndexDef, string, bool) {
	return nil, "", false
}