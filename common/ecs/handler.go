package ecs

import (
	"net/netip"
	"sync"

	"github.com/miekg/dns"
)

type ECSHandler struct {
	PublicIpAddr netip.Addr
	Access       sync.RWMutex
}

func (e *ECSHandler) GetPublicIpPrefix(mask int) netip.Prefix {
	if !e.Access.TryRLock() {
		return netip.Prefix{}
	}
	defer e.Access.RUnlock()

	if e.PublicIpAddr.IsValid() {
		return netip.PrefixFrom(e.PublicIpAddr, mask)
	}
	return netip.Prefix{}
}

func (e *ECSHandler) UpdatePublicIpAddr() (netip.Addr, bool) {
	if !e.Access.TryLock() {
		return netip.Addr{}, false
	}
	defer e.Access.Unlock()
	results := make(chan netip.Addr, len(Providers))
	var wg sync.WaitGroup

	for _, provider := range Providers {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()
			ipAddr := e.getIPFromProvider(p)
			results <- ipAddr
		}(provider)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for ipAddr := range results {
		if ipAddr.IsValid() {
			if e.PublicIpAddr == ipAddr {
				return e.PublicIpAddr, true
			}
			e.PublicIpAddr = ipAddr
			return e.PublicIpAddr, false
		}
	}
	return netip.Addr{}, false
}

func (e *ECSHandler) getIPFromProvider(provider Provider) netip.Addr {
	client := new(dns.Client)
	m := &dns.Msg{
		Question: []dns.Question{
			{
				Name:   dns.Fqdn(provider.Domain),
				Qtype:  provider.Type,
				Qclass: provider.Class,
			},
		},
	}
	resp, _, err := client.Exchange(m, provider.DNSServer)
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
