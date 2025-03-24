package common

type RequestWinners struct {
	Agency int
}

const (
	RequestWinnersIdentifier = 2
)

func (request *RequestWinners) ToBytes() []byte {
	bytes := make([]byte, 0, 2)
	bytes = append(bytes, byte(RequestWinnersIdentifier))
	bytes = append(bytes, byte(request.Agency))
	return bytes
}