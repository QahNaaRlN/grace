#!/bin/sh
set -eu

config="$(CDPATH= cd -- "$(dirname "$0")" && pwd)/grace-firewall.nft"

nft --check --file "$config"
if nft list table inet grace_input >/dev/null 2>&1; then
  nft delete table inet grace_input
fi
nft --file "$config"
