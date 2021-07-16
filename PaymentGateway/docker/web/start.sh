#!/bin/bash
nginx

/pg-docker-entrypoint.sh &
mkdir -p /root/tor/hidden_service/hsv3
chmod -R u=rwx,g=-,o=- /root/tor
chmod -R u=rwx,g=-,o=- /root/tor/hidden_service/hsv3
/tor-docker-entrypoint.sh &

function catHS {
    sleep 30
    cat /root/tor/hidden_service/hsv3/hostname
}
catHS &
while true; do sleep 30; done;
