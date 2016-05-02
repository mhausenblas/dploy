#!/bin/sh

bg_port=$(dig _buzzgen-dployex._tcp.marathon.mesos SRV | grep ^_buzzgen-dployex._tcp.marathon.mesos. | cut -d " " -f 6)
bg_addsec=$(dig _buzzgen-dployex._tcp.marathon.mesos SRV | grep ^_buzzgen-dployex._tcp.marathon.mesos. | cut -d " " -f 7)
bg_ip=$(dig _buzzgen-dployex._tcp.marathon.mesos SRV | grep ^${bg_addsec} | cut -d " " -f 5)
export 	BGSERVER=$bg_ip:$bg_port
python sanitizer.py