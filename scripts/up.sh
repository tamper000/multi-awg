#!/bin/bash
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
   echo "Error: Script must be run as root."
   exit 1
fi

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 <awg_interface> <mihomo_interface>"
    echo "Example: $0 awg0 mihomo0"
    exit 1
fi

AWG_IF="$1"
MIHOMO_IF="$2"
TABLE_ID=111
FWMARK="0x2"
RULE_PRIO=10000

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

# Disable routing to mihomo
exit 0

echo "[*] Initializing routing: $AWG_IF -> $MIHOMO_IF via fwmark"

if ! ip link show "$AWG_IF" &> /dev/null; then
    echo "[-] Error: Interface $AWG_IF not found."
    exit 1
fi

if ! ip link show "$MIHOMO_IF" &> /dev/null; then
    echo "[-] Error: Interface $MIHOMO_IF not found. (Make sure mihomo is configured to create a TUN)"
    exit 1
fi

CLIENT_SUBNET=$(ip route show dev "$AWG_IF" proto kernel scope link | awk '{print $1}' | head -n 1)

if [[ -z "$CLIENT_SUBNET" ]]; then
    echo "[-] Error: Failed to determine subnet for $AWG_IF."
    exit 1
fi
echo "[+] Detected client subnet: $CLIENT_SUBNET"

iptables -A FORWARD -i $AWG_IF -j ACCEPT
iptables -t nat -A POSTROUTING -o $MIHOMO_IF -j MASQUERADE


echo "[*] Configuring Policy-Based Routing (Table $TABLE_ID)..."
ip route replace default dev "$MIHOMO_IF" table "$TABLE_ID"
ip rule del fwmark "$FWMARK" table "$TABLE_ID" 2>/dev/null || true
ip rule add fwmark "$FWMARK" table "$TABLE_ID" priority "$RULE_PRIO"

echo "[*] Configuring iptables marking rules..."
iptables -t mangle -C PREROUTING -i "$AWG_IF" -s "$CLIENT_SUBNET" -d "$CLIENT_SUBNET" -j RETURN 2>/dev/null \
    || iptables -t mangle -I PREROUTING 1 -i "$AWG_IF" -s "$CLIENT_SUBNET" -d "$CLIENT_SUBNET" -j RETURN

iptables -t mangle -C PREROUTING -i "$AWG_IF" -s "$CLIENT_SUBNET" -j MARK --set-mark "$FWMARK" 2>/dev/null \
    || iptables -t mangle -A PREROUTING -i "$AWG_IF" -s "$CLIENT_SUBNET" -j MARK --set-mark "$FWMARK"

echo "[*] Configuring forwarding between interfaces..."
iptables -D FORWARD -i "$AWG_IF" -o "$MIHOMO_IF" -j ACCEPT 2>/dev/null || true
iptables -D FORWARD -i "$MIHOMO_IF" -o "$AWG_IF" -j ACCEPT 2>/dev/null || true
iptables -I FORWARD -i "$AWG_IF" -o "$MIHOMO_IF" -j ACCEPT
iptables -I FORWARD -i "$MIHOMO_IF" -o "$AWG_IF" -j ACCEPT

echo "[+] Done! Client traffic successfully redirected to $MIHOMO_IF via fwmark."
