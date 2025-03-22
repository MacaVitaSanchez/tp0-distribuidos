package common

import (
	"errors"
	"net"
)

func writeExact(conn net.Conn, data []byte) error {
	sentBytes := 0
	for sentBytes < len(data) {
		n, err := conn.Write(data[sentBytes:])
		if err != nil {
			return err
		}
		sentBytes += n
	}
	return nil
}

func readExact(conn net.Conn, length int) ([]byte, error) {
	data := make([]byte, length)
	readBytes := 0

	for readBytes < length {
		n, err := conn.Read(data[readBytes:])
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, errors.New("connection closed before expected data")
		}
		readBytes += n
	}

	return data, nil
}