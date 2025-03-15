package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	nombreArchivoSalida := flag.String("nombreArchivoSalida", "docker-compose-dev.yaml", "Nombre del archivo de salida")
	cantidadClientes := flag.Int("cantidadClientes", 1, "Cantidad de clientes esperada")

	flag.Parse()

	compose := fmt.Sprintf(`name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=%d
      - LOGGING_LEVEL=DEBUG
    networks:
      - testing_net

`, *cantidadClientes)

	for i := 1; i <= *cantidadClientes; i++ {
		nombreCliente := fmt.Sprintf("client%d", i)
		compose += fmt.Sprintf(`  %s:
    container_name: %s
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=%d
      - CLI_LOG_LEVEL=DEBUG
    networks:
      - testing_net
    depends_on:
      - server

`, nombreCliente, nombreCliente, i)
	}

	compose += `networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
`

	err := os.WriteFile(*nombreArchivoSalida, []byte(compose), 0644)
	if err != nil {
		fmt.Printf("Error al escribir el archivo: %v\n", err)
		return
	}
}