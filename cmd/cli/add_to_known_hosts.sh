#!/bin/bash

# For help run with -h

usage() {
	echo -e "Makes sure the pubkey returned from proxy matches ssh-keyscan of the host, and formats in a way that can be appended to known_hosts.\nUsage:\t./add_to_known_hosts.sh [--proxy=http://127.0.0.1:8080] --ssh-host=<hosts ip> [--ssh-port=22] >> ~/.ssh/known_hosts\n\tMake sure your cvm-reverse-proxy client is running."
}

PORT=22
PROXY="http://127.0.0.1:8080"

for i in "$@"
do
case $i in
    --proxy=*)
		PROXY="${i#*=}"
		;;
    --ssh-host=*)
		HOST="${i#*=}"
		;;
    --ssh-port=*)
		PORT="${i#*=}"
		;;
    -h|--help|*)
		usage
		exit 0
		;;
esac
done

if [[ -z "$HOST" ]]; then
	usage
	exit 1
fi

pubkey=`curl -s $PROXY/pubkey`
ssh-keyscan -p "$PORT" -H "$HOST" 2>/dev/null | grep "${pubkey}"
