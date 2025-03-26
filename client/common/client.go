package common

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/op/go-logging"
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

// CreateClientSocket initializes the client socket. In case of failure,
// it retries a number of times before returning an error.
func (c *Client) createClientSocket() error {
    var conn net.Conn
    var err error
    maxRetries := 5
    retryDelay := 2 * time.Second

    for i := 0; i < maxRetries; i++ {
        conn, err = net.Dial("tcp", c.config.ServerAddress)
        if err == nil {
            c.conn = conn
            return nil
        }

        log.Infof(
			"action: connect | result: in_progress | client_id: %v | error: %v",
			c.config.ID,
			err,
		)

        time.Sleep(retryDelay)
    }
    
	log.Criticalf(
		"action: connect | result: in_progress | client_id: %v | error: %v",
		c.config.ID,
		err,
	)
    return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
		// Create the connection the server in every loop iteration.
		// Before sending a message, check if SIGTERM was received to exit immediately.
		select {
		case <-c.quitChan:
			return
		default:
			c.createClientSocket()

			// TODO: Modify the send to avoid short-write
			fmt.Fprintf(
				c.conn,
				"[CLIENT %v] Message N°%v\n",
				c.config.ID,
				msgID,
			)
			msg, err := bufio.NewReader(c.conn).ReadString('\n')
			c.conn.Close()

			if err != nil {
				log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
					c.config.ID,
					err,
				)
				return
			}

			log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
				c.config.ID,
				msg,
			)

		// Waits for the next message interval, but exits immediately if SIGTERM is received.
			select {
			case <-c.quitChan:
				return
			case <-time.After(c.config.LoopPeriod):
			}
		}
	}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
