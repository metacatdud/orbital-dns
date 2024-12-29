package orbital

import (
	"github.com/miekg/dns"
	"net"
)

// Dumb implementation of dnsWriter for allowing DoH pass to DNS request
// VERY EXPERIMENTAL

type orbitalWriter struct {
	response *dns.Msg
}

func (ow *orbitalWriter) WriteMsg(msg *dns.Msg) error {
	ow.response = msg
	return nil
}

// Unused/empty methods to fulfill the dns.ResponseWriter interface implementation
func (ow *orbitalWriter) Write(b []byte) (int, error) { return 0, nil }
func (ow *orbitalWriter) Close() error                { return nil }
func (ow *orbitalWriter) TsigStatus() error           { return nil }
func (ow *orbitalWriter) TsigTimersOnly(bool)         {}
func (ow *orbitalWriter) Hijack()                     {}
func (ow *orbitalWriter) LocalAddr() net.Addr         { return nil }
func (ow *orbitalWriter) RemoteAddr() net.Addr        { return nil }
