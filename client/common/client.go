package common

import (
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/op/go-logging"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/protocol"

)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
	quitChan chan struct{}
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
		quitChan: make(chan struct{}),
	}

	// Create a signal channel to handle SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	// Goroutine to listen for SIGTERM and close quitChan
	go func() {
		<-sigChan
		close(client.quitChan)
		log.Infof("action: exit | result: success")
	}()

	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) SendBet(bet *protocol.Bet) bool{
	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed

	message := bet.ToBytes()
	select {
		case <-c.quitChan:
			return false
		default:
			c.createClientSocket()
			
		err := writeExact(c.conn, message)
		if err != nil {
			log.Errorf("action: apuesta_enviada | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err)
			return false
		}
		confirmation, err := readExact(c.conn, 1)
		if err != nil {
			log.Errorf("action: read_confirmation | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return false
		}

		if confirmation[0] == 1 {
			log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v", bet.Documento, bet.Numero)
			return true
		} else {
			log.Infof("action: apuesta_enviada | result: fail | dni: %v | numero: %v", bet.Documento, bet.Numero)
			return false
		}	
		c.conn.Close()
	}
	return false
}