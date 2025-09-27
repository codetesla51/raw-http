#!/bin/bash

exec 3<>/dev/tcp/localhost/8080

# Send multiple requests through the same connection
echo -e "GET / HTTP/1.1\r\nHost: localhost:8080\r\nConnection: keep-alive\r\n\r\n" >&3
sleep 1
echo -e "GET /time HTTP/1.1\r\nHost: localhost:8080\r\nConnection: keep-alive\r\n\r\n" >&3
sleep 1
echo -e "GET /hello HTTP/1.1\r\nHost: localhost:8080\r\nConnection: close\r\n\r\n" >&3

# Read responses
cat <&3

exec 3>&-