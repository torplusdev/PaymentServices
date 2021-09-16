#!/bin/sh

function ppchtest {
    while true; do {
        echo -en "HTTP/1.1 200 OK\r\n$(date)\r\n\r\n<h1>8080 port handler</h1>\r\n\r\n" |  nc -q 1 -l 8080; 
    }
    done 
}

ppchtest &
/pg-docker-entrypoint.sh &
/tor-docker-entrypoint.sh
