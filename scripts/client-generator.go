package main

import (
	"flag"
	"fmt"
	"os"
)

func generarConfigServer(cantidadClientes int) string {
	return fmt.Sprintf(`name: tp0
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
`, cantidadClientes)
}

func generarConfigCliente(numeroCliente int) string {
	nombreCliente := fmt.Sprintf("client%d", numeroCliente)
	return fmt.Sprintf(`  %s:
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
`, nombreCliente, nombreCliente, numeroCliente)
}

func generarConfigClientes(cantidadClientes int) string {
	configClientes := ""
	for i := 1; i <= cantidadClientes; i++ {
		configClientes += generarConfigCliente(i)
	}
	return configClientes
}

func generarConfigRedes() string {
	return `networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
`
}

func generarDockerCompose(cantidadClientes int) string {
	compose := generarConfigServer(cantidadClientes)
	compose += generarConfigClientes(cantidadClientes)
	compose += generarConfigRedes()
	return compose
}

func main() {
	nombreArchivoSalida := flag.String("nombreArchivoSalida", "docker-compose-dev.yaml", "Nombre del archivo de salida")
	cantidadClientes := flag.Int("cantidadClientes", 1, "Cantidad de clientes esperada")
	flag.Parse()

	compose := generarDockerCompose(*cantidadClientes)

	err := os.WriteFile(*nombreArchivoSalida, []byte(compose), 0644)
	if err != nil {
		fmt.Printf("Error al escribir el archivo: %v\n", err)
		return
	}
}