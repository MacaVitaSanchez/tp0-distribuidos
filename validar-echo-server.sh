#!/bin/bash

mensaje="Probando echo server..."
PUERTO_SERVER=$(grep SERVER_PORT server/config.ini | cut -d ' ' -f 3)
IP_SERVER=$(grep SERVER_IP server/config.ini | cut -d ' ' -f 3)

respuesta=$(docker run --rm --network tp0_testing_net busybox:latest sh -c "echo '$mensaje' | nc $IP_SERVER $PUERTO_SERVER")

if [ "$respuesta" = "$mensaje" ]; then
    echo "action: test_echo_server | result: success"
else
    echo "action: test_echo_server | result: fail"
fi