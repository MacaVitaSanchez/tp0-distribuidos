package protocol

import (
	"encoding/binary"
)

const (
	AgenciaLength    = 1
	NombreLength     = 1
	ApellidoLength   = 1
	DocumentoLength  = 8
	NacimientoLength = 10
	NumeroLength     = 2
	BetMessageIdentifier = 1
)

type Bet struct {
	Agencia    int
	Nombre     string
	Apellido   string
	Documento  string
	Nacimiento string
	Numero     int
}

func (bet *Bet) ToBytes() []byte {

	nombreLen := len(bet.Nombre)
	apellidoLen := len(bet.Apellido)

	payload := make([]byte, 0, AgenciaLength+NombreLength+nombreLen+ApellidoLength+apellidoLen+DocumentoLength+NacimientoLength+NumeroLength)

	payload = append(payload, byte(bet.Agencia))

	payload = append(payload, byte(nombreLen))
	payload = append(payload, []byte(bet.Nombre)...)

	payload = append(payload, byte(apellidoLen))
	payload = append(payload, []byte(bet.Apellido)...)

	payload = append(payload, []byte(bet.Documento)...)

	payload = append(payload, []byte(bet.Nacimiento)...)

	numBytes := make([]byte, NumeroLength)
	binary.BigEndian.PutUint16(numBytes, uint16(bet.Numero))
	payload = append(payload, numBytes...)

	totalLength := len(payload)
	message := make([]byte, 2+totalLength)
	binary.BigEndian.PutUint16(message, uint16(totalLength))
	copy(message[2:], payload)

	return message
}

func SerializeBetBatch(betBatch []*Bet) []byte {
	serialized := []byte{BetMessageIdentifier, byte(len(betBatch))}
   for _, bet := range betBatch {
		serialized = append(serialized, bet.ToBytes()...)
   }

   return serialized
}

func NewBet(agenciaId int, nombre string, apellido string, documento string, nacimiento string, numeroApostado int) *Bet {

	bet := &Bet{
		Agencia:    agenciaId,
		Nombre:     nombre,
		Apellido:   apellido,
		Documento:  documento,
		Nacimiento: nacimiento,
		Numero:     numeroApostado,
	}

	return bet

}