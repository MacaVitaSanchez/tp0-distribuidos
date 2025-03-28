package common

import (
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	"github.com/op/go-logging"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/protocol"
	"encoding/csv"
    "io"
	"strconv"

)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
	BatchSize int
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
		log.Infof("action: graceful_shutdown | result: in_progress | client_id: %v", config.ID)
		
		if client.conn != nil {
			client.conn.Close()
		}
		
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


func (c *Client) CreateBetsFromCSV(pathBets string, agencia int) ([][]*protocol.Bet, error) {
    file, err := os.Open(pathBets)
    if err != nil {
        log.Errorf("action: open_file | result: fail | client_id: %v | error: %v", c.config.ID, err)
        return nil, err
    }
    defer file.Close()

    reader := csv.NewReader(file)
    var allBets []*protocol.Bet

    for {
        line, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Errorf("action: read_bet | result: fail | client_id: %v | error: %v", c.config.ID, err)
            return nil, err
        }

        if len(line) != 5 {
            log.Errorf("action: read_bet | result: fail | client_id: %v | error: Insufficient data on line", c.config.ID)
            continue
        }
        numeroApostado, _ := strconv.Atoi(line[4])
        bet := protocol.NewBet(
            agencia,
            line[0],
            line[1],
            line[2],
            line[3],
            numeroApostado, 
        )

        allBets = append(allBets, bet)
    }

    var betBatches [][]*protocol.Bet
    for i := 0; i < len(allBets); i += c.config.BatchSize {
        end := i + c.config.BatchSize
        if end > len(allBets) {
            end = len(allBets)
        }
        betBatches = append(betBatches, allBets[i:end])
    }

    return betBatches, nil
}


// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) SendBet(pathBets string) bool {
    select {
    case <-c.quitChan:
        return false
    default:
    }
    
    agenciaID, _ := strconv.Atoi(c.config.ID)
    batches, err := c.CreateBetsFromCSV(pathBets, agenciaID)
    if err != nil {
        return false
    }
    
    err = c.createClientSocket()
    if err != nil {
        log.Errorf("action: send_bet | result: fail | client_id: %v | error: no se pudo conectar al servidor", c.config.ID)
        return false
    }
    
    c.conn.SetDeadline(time.Now().Add(10 * time.Second))
    
    done := make(chan struct{})
    go func() {
        select {
        case <-c.quitChan:
            c.conn.Close()
        case <-done:
        }
    }()
    
    defer func() {
        c.conn.Close()
        close(done)
    }()
    
    for _, batch := range batches {
        message := protocol.SerializeBetBatch(batch)
        select {
        case <-c.quitChan:
            return false
        default:
        
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
                    err)
            return false
        }

        if confirmation[0] == 1 {
                log.Infof("action: apuesta_enviada | result: success | batch_size: %v", len(batch))
        } else {
                log.Infof("action: apuesta_enviada | result: fail | batch_size: %v", len(batch))
            return false
        }
    }
    
    }
    time.Sleep(1 * time.Second) // sleep para que el servidor pueda imprimir todas las validaciones en el logger
    return true
}

func (c *Client) GetWinners(agencia int) {
	select {
	case <-c.quitChan:
		return
	default:
	}

	err := c.createClientSocket()
	if err != nil {
		log.Errorf("action: consulta_ganadores | result: fail | client_id: %v | error: no se pudo conectar al servidor", c.config.ID)
		return
	}
	
	c.conn.SetDeadline(time.Now().Add(5 * time.Second))
	
	done := make(chan struct{})
	go func() {
		select {
		case <-c.quitChan:
			c.conn.Close()
		case <-done:
		}
	}()
	
	defer func() {
		c.conn.Close()
		close(done)
	}()

	request := RequestWinners{
		Agency: agencia,
	}

	serializedRequest := request.ToBytes()
	if err := writeExact(c.conn, serializedRequest); err != nil {
		log.Errorf("action: serialize_request | result: fail | client_id: %v | error %v", c.config.ID, err)
		return
	}

	winners, err := DeserializeWinners(c.conn)
	if err != nil {
		log.Errorf("action: consulta_ganadores | result: fail | client_id: %v | error %v", c.config.ID, err)
		return
	}

	log.Infof("action: consulta_ganadores | result: success | client_id: %v | cant_ganadores: %d", c.config.ID, len(winners.Dnis))
}

