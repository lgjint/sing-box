package route

import (
	"net/netip"
	"sync"

	"github.com/miekg/dns"
)

var (
	cachedPublicIpAddr netip.Addr
	cacheMutex         sync.RWMutex
)

type IPService struct {
	Name      string
	DNSServer string
	Domain    string
	Type      uint16
	Class     uint16
	ParseFunc func([]string) netip.Addr
}

var IPServices = []IPService{
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

func getPublicIpAddr() netip.Addr {
	if !cacheMutex.TryRLock() {
		return netip.Addr{}
	}
	defer cacheMutex.RUnlock()

	if cachedPublicIpAddr.IsValid() {
		return cachedPublicIpAddr
	}
	return netip.Addr{}
}

func updatePublicIpAddr() (netip.Addr, bool) {
	type result struct {
		ipAddr  netip.Addr
		service IPService
	}
	if !cacheMutex.TryLock() {
		return netip.Addr{}, false
	}
	defer cacheMutex.Unlock()
	results := make(chan netip.Addr, len(IPServices))
	var wg sync.WaitGroup

	for _, service := range IPServices {
		wg.Add(1)
		go func(s IPService) {
			defer wg.Done()
			ipAddr := getIPFromService(s)
			results <- ipAddr
		}(service)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for ipAddr := range results {
		if ipAddr.IsValid() {
			if cachedPublicIpAddr == ipAddr {
				return cachedPublicIpAddr, true
			}
			cachedPublicIpAddr = ipAddr
			return cachedPublicIpAddr, false
		}
	}
	return netip.Addr{}, false
}

func getIPFromService(service IPService) netip.Addr {
	client := new(dns.Client)
	m := &dns.Msg{
		Question: []dns.Question{
			{
				Name:   dns.Fqdn(service.Domain),
				Qtype:  service.Type,
				Qclass: service.Class,
			},
		},
	}
	resp, _, err := client.Exchange(m, service.DNSServer)
	if err != nil {
		return netip.Addr{}
	}

	for _, answer := range resp.Answer {
		switch answer := answer.(type) {
		case *dns.A:
			ipAddr, ok := netip.AddrFromSlice(answer.A)
			if ok {
				return ipAddr
			}
		case *dns.AAAA:
			ipAddr, ok := netip.AddrFromSlice(answer.AAAA)
			if ok {
				return ipAddr
			}
		case *dns.TXT:
			ipAddr, err := netip.ParseAddr(answer.Txt[0])
			if err == nil {
				return ipAddr
			}
		}
	}
	return netip.Addr{}
}
