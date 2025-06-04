#!/bin/sh

# Usage: ./add_to_known_hosts.sh <http proxy> <host ip> >> ~/.ssh/known_hosts
# Makes sure the pubkey returned from proxy matches ssh-keyscan of the host, and formats in a way that can be appended to known_hosts

if [ $1 = "-h" ]; then
	echo "Usage: ./add_to_known_hosts.sh <http proxy> <host ip> >> ~/.ssh/known_hosts (or append manually)"
	exit 0;
fi

pubkeys=`curl -s $1/pubkey`
host_keys=`ssh-keyscan -H "$2" 2>/dev/null; ssh-keyscan -H -p 10022 "$2" 2>/dev/null`

echo "$pubkeys" | while IFS= read -r pubkey; do
	if [ -n "$pubkey" ]; then
		echo "$host_keys" | grep "${pubkey}"
	fi
done