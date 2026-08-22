package network

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/miekg/dns"
)

const dnsStateDir = "/var/run/minidocker/dns"

type DNSRecordsMap map[string]string // hostname -> IP

var mu sync.Mutex

func ensureStateDir() {
	_ = os.MkdirAll(dnsStateDir, 0755)
}

func getTablePath(gatewayIP string) string {
	return filepath.Join(dnsStateDir, fmt.Sprintf("%s.json", gatewayIP))
}

func getPidPath(gatewayIP string) string {
	return filepath.Join(dnsStateDir, fmt.Sprintf("%s.pid", gatewayIP))
}

func RegisterRecord(gatewayIP, name, ip string) {
	if gatewayIP == "" || name == "" || ip == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	ensureStateDir()

	records := loadTable(gatewayIP)
	clean := strings.ToLower(strings.TrimSuffix(name, ".")) + "."
	records[clean] = ip
	saveTable(gatewayIP, records)
}

func UnregisterRecord(gatewayIP, name string) {
	if gatewayIP == "" || name == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	ensureStateDir()

	records := loadTable(gatewayIP)
	clean := strings.ToLower(strings.TrimSuffix(name, ".")) + "."
	delete(records, clean)
	saveTable(gatewayIP, records)
}

func loadTable(gatewayIP string) DNSRecordsMap {
	table := make(DNSRecordsMap)
	data, err := os.ReadFile(getTablePath(gatewayIP))
	if err == nil {
		_ = json.Unmarshal(data, &table)
	}
	return table
}

func saveTable(gatewayIP string, table DNSRecordsMap) {
	data, err := json.Marshal(table)
	if err == nil {
		_ = os.WriteFile(getTablePath(gatewayIP), data, 0644)
	}
}

// StartEmbeddedDNS asegura que exista un proceso daemon independiente escuchando en gatewayIP:53
func StartEmbeddedDNS(gatewayIP string) {
	ensureStateDir()
	pidFile := getPidPath(gatewayIP)

	// Comprobar si el daemon ya está vivo
	if data, err := os.ReadFile(pidFile); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			if syscall.Kill(pid, 0) == nil {
				return // Ya está corriendo y saludable
			}
		}
	}

	// Lanzar el subproceso demonizado desacoplado de la terminal
	cmd := exec.Command("/proc/self/exe", "__dnsd__", gatewayIP)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Crea una sesión independiente; inmune a señales de la terminal
	}

	if err := cmd.Start(); err == nil {
		_ = os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
		_ = cmd.Process.Release()
	}
}

// RunDNSDaemon es el bucle del servidor DNS ejecutado por el daemon
func RunDNSDaemon(gatewayIP string) error {
	listenAddr := fmt.Sprintf("%s:53", gatewayIP)

	mux := dns.NewServeMux()
	mux.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true

		records := loadTable(gatewayIP)

		for _, q := range r.Question {
			target := strings.ToLower(strings.TrimSuffix(q.Name, ".")) + "."
			if ip, exists := records[target]; exists {
				if q.Qtype == dns.TypeA {
					rr, err := dns.NewRR(fmt.Sprintf("%s 60 IN A %s", q.Name, ip))
					if err == nil {
						m.Answer = append(m.Answer, rr)
					}
				}
				m.SetRcode(r, dns.RcodeSuccess)
				_ = w.WriteMsg(m)
				return
			}
		}

		// Reenvío a 8.8.8.8 para dominios externos
		c := new(dns.Client)
		in, _, err := c.Exchange(r, "8.8.8.8:53")
		if err == nil {
			_ = w.WriteMsg(in)
			return
		}

		m.SetRcode(r, dns.RcodeNameError)
		_ = w.WriteMsg(m)
	})

	srv := &dns.Server{
		Addr:    listenAddr,
		Net:     "udp",
		Handler: mux,
	}

	return srv.ListenAndServe()
}
