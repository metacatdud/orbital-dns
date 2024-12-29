package main

import (
	"fmt"
	"log"
	"orbitaldns/orbital"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
func run() error {
	ipv6Address := "-"
	srv := orbital.NewOrbitalDNS("ygg0", "certs/cert.pem", "certs/key.pem")

	// Add a test zone
	srv.Zone().AddRecord("hello.orbital", ipv6Address)

	log.Fatal(srv.Start())
	return nil
}
