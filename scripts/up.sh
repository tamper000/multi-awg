#!/bin/bash
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
   echo "Error: Script must be run as root."
   exit 1
fi

if [[ $# -ne 1 ]]; then
    echo "Usage: $0 <awg_interface>"
    echo "Example: $0 awg0"
    exit 1
fi

AWG_IF="$1"

echo "[*] Initializing AmneziaWG: $AWG_IF "

amneziawg-go $AWG_IF
awg syncconf $AWG_IF <(awg-quick strip $AWG_IF)

AWG_ADDR=$(awk -F'=' '/^[[:space:]]*Address[[:space:]]*=/{gsub(/[[:space:]]/,"",$2); print $2; exit}' /etc/amnezia/amneziawg/$AWG_IF.conf)
AWG_MTU=$(awk -F'=' '/^[[:space:]]*MTU[[:space:]]*=/{gsub(/[[:space:]]/,"",$2); print $2; exit}' /etc/amnezia/amneziawg/$AWG_IF.conf)

if [[ -z "$AWG_ADDR" ]]; then
    echo "[-] Error: Address not found in /etc/amnezia/amneziawg/$AWG_IF.conf"
    exit 1
fi
if [[ -z "$AWG_MTU" ]]; then
    AWG_MTU=1420
fi
ip -4 address add "$AWG_ADDR" dev "$AWG_IF"
ip link set mtu "$AWG_MTU" up dev "$AWG_IF"
ip link set dev "$AWG_IF" up

# Direct NAT through the container's external interface.
iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE 2>/dev/null || true
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE

# Allow AWG clients out and only established traffic back in.
iptables -D FORWARD -i "$AWG_IF" -o eth0 -j ACCEPT 2>/dev/null || true
iptables -D FORWARD -i eth0 -o "$AWG_IF" -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT 2>/dev/null || true
iptables -I FORWARD 1 -i "$AWG_IF" -o eth0 -j ACCEPT
iptables -I FORWARD 1 -i eth0 -o "$AWG_IF" -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

# Drop invalid conntrack packets before forwarding.
iptables -D FORWARD -m conntrack --ctstate INVALID -j DROP 2>/dev/null || true
iptables -I FORWARD 2 -m conntrack --ctstate INVALID -j DROP

exit 0
