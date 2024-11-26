package ecs

import (
	"net/netip"

	"github.com/miekg/dns"
)

type Provider struct {
	Name      string
	DNSServer string
	Domain    string
	Type      uint16
	Class     uint16
	ParseFunc func([]string) netip.Addr
}

var Providers = []Provider{
	{
		Name:      "Cloudflare",
		DNSServer: "1.1.1.1:53",
		Domain:    "whoami.cloudflare.",
		Type:      dns.TypeTXT,
		Class:     dns.ClassCHAOS,
	},
	{
		Name:      "OpenDNS",
		DNSServer: "208.67.222.222:53",
		Domain:    "myip.opendns.com",
		Type:      dns.TypeA,
		Class:     dns.ClassINET,
	},
}
