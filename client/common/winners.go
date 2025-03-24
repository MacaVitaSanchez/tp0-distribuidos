package common

import (
	"encoding/binary"
	"net"
)

type Winners struct {
	Dnis []string
}

func DeserializeWinners(socket net.Conn) (*Winners, error) {
	var winners Winners

	var numDocuments uint16
	err := binary.Read(socket, binary.BigEndian, &numDocuments)
	if err != nil {
		return nil, err
	}

	for i := 0; i < int(numDocuments); i++ {
		dni, err := readExact(socket, 8)
		if err != nil {
			return nil, err
		}
		winners.Dnis = append(winners.Dnis, string(dni))
	}

	return &winners, nil
}