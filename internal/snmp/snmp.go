package snmp

import (
	//"crypto/tls"
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
)

type Version string

const (
	V2c Version = "2c"
	V3  Version = "3"
)

type V3SecLevel string

const (
	NoAuthNoPriv V3SecLevel = "noAuthNoPriv"
	AuthNoPriv   V3SecLevel = "authNoPriv"
	AuthPriv     V3SecLevel = "authPriv"
)

type Config struct {
	Target    string
	Port      uint16
	Timeout   time.Duration
	Retries   int
	Version   Version
	Community string // v2c

	// v3
	SecLevel   V3SecLevel
	Username   string
	AuthProto  string
	AuthPass   string
	PrivProto  string
	PrivPass   string
}

func New(cfg Config) (*gosnmp.GoSNMP, error) {
	s := &gosnmp.GoSNMP{
		Target:  cfg.Target,
		Port:    cfg.Port,
		Timeout: cfg.Timeout,
		Retries: cfg.Retries,
		// Some devices are picky; keep defaults safe:
		MaxOids:  gosnmp.MaxOids,
		Transport: "udp",
		// TlsConfig: &tls.Config{InsecureSkipVerify: true},
	}

	switch cfg.Version {
	case V2c:
		s.Version = gosnmp.Version2c
		s.Community = cfg.Community
	case V3:
		s.Version = gosnmp.Version3
		usm := &gosnmp.UsmSecurityParameters{
			UserName: cfg.Username,
		}

		// Auth protocol
		switch cfg.AuthProto {
		case "MD5":
			usm.AuthenticationProtocol = gosnmp.MD5
		case "SHA":
			usm.AuthenticationProtocol = gosnmp.SHA
		case "SHA-224":
			usm.AuthenticationProtocol = gosnmp.SHA224
		case "SHA-256":
			usm.AuthenticationProtocol = gosnmp.SHA256
		case "SHA-384":
			usm.AuthenticationProtocol = gosnmp.SHA384
		case "SHA-512":
			usm.AuthenticationProtocol = gosnmp.SHA512
		case "":
			usm.AuthenticationProtocol = gosnmp.NoAuth
		default:
			return nil, fmt.Errorf("unsupported auth protocol: %q", cfg.AuthProto)
		}
		usm.AuthenticationPassphrase = cfg.AuthPass

		// Priv protocol
		switch cfg.PrivProto {
		case "DES":
			usm.PrivacyProtocol = gosnmp.DES
		case "AES":
			usm.PrivacyProtocol = gosnmp.AES
		case "AES-192":
			usm.PrivacyProtocol = gosnmp.AES192
		case "AES-256":
			usm.PrivacyProtocol = gosnmp.AES256
		case "":
			usm.PrivacyProtocol = gosnmp.NoPriv
		default:
			return nil, fmt.Errorf("unsupported priv protocol: %q", cfg.PrivProto)
		}
		usm.PrivacyPassphrase = cfg.PrivPass

		s.SecurityParameters = usm

		// Security level
		switch cfg.SecLevel {
		case NoAuthNoPriv:
			s.MsgFlags = gosnmp.NoAuthNoPriv
		case AuthNoPriv:
			s.MsgFlags = gosnmp.AuthNoPriv
		case AuthPriv:
			s.MsgFlags = gosnmp.AuthPriv
		default:
			return nil, fmt.Errorf("unsupported security level: %q", cfg.SecLevel)
		}
		s.SecurityModel = gosnmp.UserSecurityModel
	default:
		return nil, fmt.Errorf("unsupported snmp version: %q", cfg.Version)
	}

	return s, nil
}
