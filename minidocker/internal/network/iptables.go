package network

import (
	"fmt"
	"os/exec"
)

// AddPortForwarding configura el reenvío DNAT en iptables
func AddPortForwarding(hostPort, containerPort int, containerIP string) error {
	dest := fmt.Sprintf("%s:%d", containerIP, containerPort)
	hPortStr := fmt.Sprintf("%d", hostPort)
	cPortStr := fmt.Sprintf("%d", containerPort)

	_ = exec.Command("iptables", "-t", "nat", "-A", "PREROUTING", "-p", "tcp", "-m", "tcp", "--dport", hPortStr, "-j", "DNAT", "--to-destination", dest).Run()
	_ = exec.Command("iptables", "-t", "nat", "-A", "OUTPUT", "-p", "tcp", "-m", "tcp", "--dport", hPortStr, "-j", "DNAT", "--to-destination", dest).Run()
	_ = exec.Command("iptables", "-A", "FORWARD", "-p", "tcp", "-d", containerIP, "--dport", cPortStr, "-j", "ACCEPT").Run()
	_ = exec.Command("iptables", "-A", "FORWARD", "-m", "state", "--state", "ESTABLISHED,RELATED", "-j", "ACCEPT").Run()
	_ = exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-p", "tcp", "-d", containerIP, "--dport", cPortStr, "-j", "MASQUERADE").Run()

	return nil
}

// RemovePortForwarding limpia las reglas iptables generadas
func RemovePortForwarding(hostPort, containerPort int, containerIP string) {
	dest := fmt.Sprintf("%s:%d", containerIP, containerPort)
	hPortStr := fmt.Sprintf("%d", hostPort)
	cPortStr := fmt.Sprintf("%d", containerPort)

	_ = exec.Command("iptables", "-t", "nat", "-D", "PREROUTING", "-p", "tcp", "-m", "tcp", "--dport", hPortStr, "-j", "DNAT", "--to-destination", dest).Run()
	_ = exec.Command("iptables", "-t", "nat", "-D", "OUTPUT", "-p", "tcp", "-m", "tcp", "--dport", hPortStr, "-j", "DNAT", "--to-destination", dest).Run()
	_ = exec.Command("iptables", "-D", "FORWARD", "-p", "tcp", "-d", containerIP, "--dport", cPortStr, "-j", "ACCEPT").Run()
	_ = exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING", "-p", "tcp", "-d", containerIP, "--dport", cPortStr, "-j", "MASQUERADE").Run()
}
