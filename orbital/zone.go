package orbital

import (
	"github.com/miekg/dns"
)

type Zone struct {
	records map[string]string
}

func NewZone() *Zone {
	return &Zone{
		records: make(map[string]string),
	}
}

func (z *Zone) AddRecord(name, ip string) {
	z.records[dns.Fqdn(name)] = ip
}

func (z *Zone) Lookup(name string) (string, bool) {
	ip, found := z.records[dns.Fqdn(name)]
	return ip, found
}
