package yandex360api

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

func (y *Yandex360ApiMock) handleDNSRequest(w dns.ResponseWriter, req *dns.Msg) {
	fmt.Println("\n\nHandleDNS")
	msg := new(dns.Msg)
	fmt.Println("\n\n>" + msg.String())
	msg.SetReply(req)
	switch req.Opcode {
	case dns.OpcodeQuery:
		for _, q := range msg.Question {
			fmt.Println("\n\n>> for")
			if err := y.addDNSAnswer(q, msg, req); err != nil {
				msg.SetRcode(req, dns.RcodeServerFailure)
				break
			}
		}
	}
	w.WriteMsg(msg)
}

func (y *Yandex360ApiMock) addDNSAnswer(q dns.Question, msg *dns.Msg, req *dns.Msg) error {
	switch q.Qtype {
	// Always return loopback for any A query
	case dns.TypeA:
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN A 127.0.0.1", q.Name))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		return nil

	// TXT records are the only important record for ACME dns-01 challenges
	case dns.TypeTXT:
		fmt.Println("\n\n>> TypeTXT: " + q.Name)

		found := false
		var records Records
		var requestedSubDomain string

	domainLookup:
		for _, orgDomains := range y.settings.organizationsAndDomains {
			for dom, rec := range orgDomains {
				if strings.HasSuffix(q.Name, "."+dom+".") {
					records = rec
					found = true
					requestedSubDomain = strings.TrimSuffix(q.Name, "."+dom+".")
					break domainLookup
				}
			}
		}

		if !found {
			fmt.Println("\n\n>> !FOUND DOMAIN")
			msg.SetRcode(req, dns.RcodeNameError)
			return nil
		}

		var dnsRecord DnsRecord
		found = false
		for _, record := range records {
			if record.Name == requestedSubDomain && record.Type == "TXT" {
				dnsRecord = record
				found = true
				break
			}
		}

		if !found {
			fmt.Println("\n\n>> !FOUND TXT subdomain " + requestedSubDomain)
			msg.SetRcode(req, dns.RcodeNameError)
			return nil
		}

		fmt.Println("\n\n>> FOUND: " + fmt.Sprintf("%s 5 IN TXT %s", q.Name, dnsRecord.Text))
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN TXT %s", q.Name, dnsRecord.Text))
		if err != nil {
			fmt.Println("\n\n>> rrErr " + err.Error())
			return err
		}
		fmt.Println("\n\n>> answer")
		msg.Answer = append(msg.Answer, rr)
		return nil

	// NS and SOA are for authoritative lookups, return obviously invalid data
	case dns.TypeNS:
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN NS ns.example-acme-webook.invalid.", q.Name))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		return nil
	case dns.TypeSOA:
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN SOA %s 20 5 5 5 5", "ns.example-acme-webook.invalid.", "ns.example-acme-webook.invalid."))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		return nil
	default:
		return fmt.Errorf("unimplemented record type %v", q.Qtype)
	}
}
