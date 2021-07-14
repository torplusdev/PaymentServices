nginx

/pg-docker-entrypoint.sh &
/tor-docker-entrypoint.sh &

function catHS {
    sleep 30
    cat /root/tor/hidden_service/hsv3/hostname
}
catHS &
while true; do sleep 30; done;
