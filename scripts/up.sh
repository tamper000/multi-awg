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
CONF="/etc/amnezia/amneziawg/$AWG_IF.conf"

echo "[*] Initializing AmneziaWG: $AWG_IF"

amneziawg-go "$AWG_IF"
awg syncconf "$AWG_IF" <(awg-quick strip "$AWG_IF")

AWG_ADDR=$(awk -F'=' '
    /^[[:space:]]*Address[[:space:]]*=/ {
        gsub(/[[:space:]]/, "", $2)
        print $2
        exit
    }
' "$CONF")

AWG_MTU=$(awk -F'=' '
    /^[[:space:]]*MTU[[:space:]]*=/ {
        gsub(/[[:space:]]/, "", $2)
        print $2
        exit
    }
' "$CONF")

if [[ -z "$AWG_ADDR" ]]; then
    echo "[-] Error: Address not found in $CONF"
    exit 1
fi

if [[ -z "$AWG_MTU" ]]; then
    AWG_MTU=1280
fi

ip -4 address add "$AWG_ADDR" dev "$AWG_IF"
ip link set mtu "$AWG_MTU" dev "$AWG_IF"
ip link set up dev "$AWG_IF"

echo "[*] Configuring IPv4 NAT"

iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE 2>/dev/null || true
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE

echo "[*] Configuring forwarding"

iptables -D FORWARD -i "$AWG_IF" -o eth0 -j ACCEPT 2>/dev/null || true
iptables -D FORWARD -i eth0 -o "$AWG_IF" -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT 2>/dev/null || true
iptables -I FORWARD 1 -i "$AWG_IF" -o eth0 -j ACCEPT
iptables -I FORWARD 1 -i eth0 -o "$AWG_IF" -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

# Remove the old INVALID rule if it was added by a previous version.
iptables -D FORWARD -m conntrack --ctstate INVALID -j DROP 2>/dev/null || true

echo "[*] Configuring TCP MSS clamping"

iptables -t mangle -D FORWARD -i "$AWG_IF" -o eth0 -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null || true
iptables -t mangle -D FORWARD -i eth0 -o "$AWG_IF" -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null || true
iptables -t mangle -I FORWARD 1 -i "$AWG_IF" -o eth0 -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu
iptables -t mangle -I FORWARD 1 -i eth0 -o "$AWG_IF" -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu

echo "[+] AmneziaWG is ready"
echo "[+] Interface: $AWG_IF"
echo "[+] Address: $AWG_ADDR"
echo "[+] MTU: $AWG_MTU"
