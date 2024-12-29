package orbital

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/miekg/dns"
	"io"
	"log"
	"net"
	"net/http"
)

type OrbitalDNS struct {
	NetworkInterface string
	CertPath         string
	KeyPath          string

	zone        *Zone
	upstream    string
	ipv6Address string
}

func NewOrbitalDNS(networkInterface, certPath, certKeyPath string) *OrbitalDNS {
	return &OrbitalDNS{
		NetworkInterface: networkInterface,
		CertPath:         certPath,
		KeyPath:          certKeyPath,
		zone:             NewZone(),
		upstream:         "1.1.1.1:53", // Hardcoded for now
	}
}

func (o *OrbitalDNS) DoHURL() string {
	return fmt.Sprintf("https://%s/dns-query", o.ipv6Address)
}

func (o *OrbitalDNS) Zone() *Zone {
	return o.zone
}

func (o *OrbitalDNS) Start() error {
	iNetworkName, err := net.InterfaceByName(o.NetworkInterface)
	if err != nil {
		return err
	}

	addrs, err := iNetworkName.Addrs()
	if err != nil {
		return err
	}

	ipv6 := ""
	for _, addr := range addrs {
		ip, _, _ := net.ParseCIDR(addr.String())
		if ip != nil && ip.To16() != nil {
			ipv6 = ip.String()
			break
		}
	}

	if ipv6 == "" {
		return fmt.Errorf("no ipv6 address found on interfce %s", o.NetworkInterface)
	}

	o.ipv6Address = ipv6
	log.Printf("[OrbitalDNS] Starting DNS\nInterface: %s \nIPv6 on %s\n", o.NetworkInterface, o.ipv6Address)

	dns.HandleFunc(".", o.handleDNSRequest)
	dnsAddr := net.JoinHostPort(o.ipv6Address, "53")
	go func() {
		dnsServer := &dns.Server{
			Addr: dnsAddr,
			Net:  "udp",
		}
		log.Printf("[OrbitalDNS] DNS running on: %s (UDP)", dnsAddr)
		if err = dnsServer.ListenAndServe(); err != nil {
			log.Fatalf("[OrbitalDNS] Failed to start DNS server: %s", err)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/dns-query", o.handleDoHRequest)
	mux.HandleFunc("/zones", o.handleDoHRequest)

	cert, err := tls.LoadX509KeyPair(o.CertPath, o.KeyPath)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	dohAddr := net.JoinHostPort(o.ipv6Address, "443")
	dohServer := &http.Server{
		Addr:      dohAddr,
		TLSConfig: tlsConfig,
		Handler:   mux,
	}

	log.Printf("[OrbitalDNS] DoH server running on %s (TLS)", dohAddr)
	return dohServer.ListenAndServeTLS("", "")

}

// handleDNSRequest handles traditional DNS requests of :53
func (o *OrbitalDNS) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	for _, q := range r.Question {
		switch q.Qtype {
		case dns.TypeAAAA:
			ip, found := o.zone.Lookup(q.Name)
			if found {
				m.Answer = append(m.Answer, &dns.AAAA{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeAAAA,
						Class:  dns.ClassINET,
						Ttl:    300,
					},
					AAAA: net.ParseIP(ip),
				})
			} else {
				// Forward unknown queries
				forw, err := o.forwardQuery(r, o.upstream)
				if err == nil && forw != nil && len(forw.Answer) > 0 {
					m.Answer = append(m.Answer, forw.Answer...)
				}
			}
		}
	}
}

// handleAddZone allow adding new zones over http requests
// Example request: {"domain": "app.orbital", "ipv6": "204:XXXX:YYYY:..."}
func (o *OrbitalDNS) handleAddZone(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid method.", http.StatusMethodNotAllowed)
		return
	}

	var zonePayload struct {
		Domain string `json:"domain"`
		IPv6   string `json:"ipv6"`
	}

	if err := json.NewDecoder(r.Body).Decode(&zonePayload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if zonePayload.Domain == "" || zonePayload.IPv6 == "" {
		http.Error(w, "Missing domain or ipv6", http.StatusBadRequest)
		return
	}

	o.zone.AddRecord(zonePayload.Domain, zonePayload.IPv6)
	log.Printf("[OrbitalDNS] Added new recor: %s -> %s\n", zonePayload.Domain, zonePayload.IPv6)

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Record added: %s -> %s\n", zonePayload.Domain, zonePayload.IPv6)
}

// handleDoHRequest handles DNS-over-HTTP requests
func (o *OrbitalDNS) handleDoHRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Only POST allowed for DoH", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	m := new(dns.Msg)
	if err = m.Unpack(body); err != nil {
		http.Error(w, fmt.Sprintf("Invalid DNS-DoH reqeust: %s", err.Error()), http.StatusBadRequest)
		return
	}

	rec := &orbitalWriter{}
	o.handleDNSRequest(rec, m)

	if rec.response == nil {
		http.Error(w, "No response from server", http.StatusBadRequest)
		return
	}

	packet, err := rec.response.Pack()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to pack DNS-DoH response: %s", err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/dns-message")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(packet)
}

func (o *OrbitalDNS) forwardQuery(msg *dns.Msg, upstream string) (*dns.Msg, error) {
	client := &dns.Client{}
	in, _, err := client.Exchange(msg, upstream)

	return in, err
}
