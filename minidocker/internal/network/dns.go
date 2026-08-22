package network

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

const dnsStoreDir = "/var/lib/minidocker/dns"

type EmbeddedDNS struct {
	server *dns.Server
}

var (
	dnsRegistry = make(map[string]*EmbeddedDNS) // gatewayIP -> servidor (solo dentro de ESTE proceso)
	registryMu  sync.Mutex

	fileMu sync.Mutex
)

func recordsFilePath(gatewayIP string) string {
	safe := strings.NewReplacer(".", "_", ":", "_").Replace(gatewayIP)
	return filepath.Join(dnsStoreDir, safe+".json")
}

func loadRecords(gatewayIP string) map[string]string {
	fileMu.Lock()
	defer fileMu.Unlock()

	data, err := os.ReadFile(recordsFilePath(gatewayIP))
	if err != nil {
		return map[string]string{}
	}
	var records map[string]string
	if err := json.Unmarshal(data, &records); err != nil {
		return map[string]string{}
	}
	return records
}

func saveRecords(gatewayIP string, records map[string]string) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	if err := os.MkdirAll(dnsStoreDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	tmp := recordsFilePath(gatewayIP) + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, recordsFilePath(gatewayIP))
}

func RegisterRecord(gatewayIP, name, ip string) {
	if gatewayIP == "" || name == "" || ip == "" {
		return
	}
	clean := strings.ToLower(strings.TrimSuffix(name, ".")) + "."

	records := loadRecords(gatewayIP)
	records[clean] = ip
	_ = saveRecords(gatewayIP, records)
}

func UnregisterRecord(gatewayIP, name string) {
	if gatewayIP == "" || name == "" {
		return
	}
	clean := strings.ToLower(strings.TrimSuffix(name, ".")) + "."

	records := loadRecords(gatewayIP)
	if _, ok := records[clean]; ok {
		delete(records, clean)
		_ = saveRecords(gatewayIP, records)
	}
}

// StartEmbeddedDNS intenta hacer bind real y síncrono en gatewayIP:53.
// Si otro proceso ya tiene ese puerto (otro contenedor de la misma red
// arrancado antes), simplemente no hace nada: ese otro proceso seguirá
// respondiendo, leyendo del mismo almacén en disco.
func StartEmbeddedDNS(gatewayIP string) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := dnsRegistry[gatewayIP]; exists {
		return
	}

	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:53", gatewayIP))
	if err != nil {
		return
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		// Ya hay un servidor DNS real en otro proceso para este gateway.
		return
	}

	instance := &EmbeddedDNS{}
	mux := dns.NewServeMux()
	mux.HandleFunc(".", handleDNSRequest)

	instance.server = &dns.Server{
		PacketConn: conn,
		Net:        "udp",
		Handler:    mux,
	}

	dnsRegistry[gatewayIP] = instance

	go func() {
		_ = instance.server.ActivateAndServe()
	}()
}

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	gatewayIP := w.LocalAddr().String()
	if host, _, err := net.SplitHostPort(gatewayIP); err == nil {
		gatewayIP = host
	}

	records := loadRecords(gatewayIP)

	for _, q := range r.Question {
		target := strings.ToLower(strings.TrimSuffix(q.Name, ".")) + "."
		if ip, found := records[target]; found {
			if q.Qtype == dns.TypeA {
				if rr, err := dns.NewRR(fmt.Sprintf("%s 60 IN A %s", q.Name, ip)); err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
			m.SetRcode(r, dns.RcodeSuccess)
			_ = w.WriteMsg(m)
			return
		}
	}

	c := new(dns.Client)
	in, _, err := c.Exchange(r, "8.8.8.8:53")
	if err == nil {
		_ = w.WriteMsg(in)
		return
	}

	m.SetRcode(r, dns.RcodeNameError)
	_ = w.WriteMsg(m)
}
