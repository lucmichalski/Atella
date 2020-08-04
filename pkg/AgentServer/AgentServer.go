package AgentServer

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	"../AgentConfig"
	"../Database"
	"../Logger"
)

type clientParams struct {
	canTalk                 bool
	id                      uint64
	currentClientHostname   string
	currentClientVectorJson string
}

type ServerClient struct {
	conn   net.Conn
	Server *server
	params clientParams
}

type server struct {
	address string
	port    int16
	config  *tls.Config
}

var (
	global uint64              = 0
	conf   *AgentConfig.Config = nil
)

// Read client data from channel
func (c *ServerClient) listen() {
	c.Server.OnNewClient(c)
	reader := bufio.NewReader(c.conn)
	var exit = false
	for exit == false {
		message, err := reader.ReadString('\n')
		if err != nil {
			c.conn.Close()
			c.Server.OnClientConnectionClosed(c, err)
			return
		}
		exit = c.Server.OnNewMessage(c, message)
	}
	c.conn.Close()
	c.Server.OnClientConnectionClosed(c, nil)
	return
}

func (c *ServerClient) Send(message string) error {
	_, err := c.conn.Write([]byte(message))
	return err
}

func (c *ServerClient) SendBytes(b []byte) error {
	_, err := c.conn.Write(b)
	return err
}

func (c *ServerClient) Conn() net.Conn {
	return c.conn
}

func (c *ServerClient) Close() error {
	return c.conn.Close()
}

func (s *server) OnNewClient(c *ServerClient) {
	Logger.LogInfo(fmt.Sprintf("New connect [%d], can talk with him - %t", global, c.params.canTalk))
	c.Send("Meow?\n")
	c.params.id = global
	global = global + 1
	if global == math.MaxInt64 {
		global = 0
	}
	c.params.currentClientHostname = ""
	c.params.currentClientVectorJson = ""
}

func (s *server) OnClientConnectionClosed(c *ServerClient, err error) {
	Logger.LogInfo(fmt.Sprintf("Client [%d] go away", c.params.id))
}

func (s *server) OnNewMessage(c *ServerClient, message string) bool {
	var (
		msg     = strings.TrimRight(message, "\r\n")
		msg_map = strings.Split(msg, " ")
	)
	switch msg_map[0] {
	case "quit", "exit":
		c.Send(fmt.Sprintf("Bye!\n"))
		return true
	}
	if c.params.canTalk == true {
		Logger.LogInfo(fmt.Sprintf("Server receive [%s]", msg))
		switch msg_map[0] {
		case "who":
			c.Send(fmt.Sprintf("Id: %d\n", c.params.id))
		case "host":
			c.params.currentClientHostname = msg_map[1]
			c.Send(fmt.Sprintf("ack %s\n", c.params.currentClientHostname))
		case "hostname":
			c.Send(fmt.Sprintf("%s\n", conf.Agent.Hostname))
		case "vector":
			c.Send(fmt.Sprintf("%s\n", AgentConfig.GetJsonVector()))
			AgentConfig.PrintJsonVector()
		case "mastervector":
			c.Send(fmt.Sprintf("%s\n", AgentConfig.GetJsonMasterVector()))
			AgentConfig.PrintJsonMasterVector()
		case "set":
			c.params.currentClientVectorJson = msg_map[1]
		}
	} else if msg == "Meow!" {
		Logger.LogInfo(fmt.Sprintf("Receive [%s], set canTalk -> true", msg))
		c.Send("canTalk\n")
		c.params.canTalk = true
	} else {
		Logger.LogInfo(fmt.Sprintf("Receive [%s], can't talk - ignore", msg))
	}
	return false
}

func (s *server) Listen() {
	var listener net.Listener
	var err error
	if s.config == nil {
		listener, err = net.Listen("tcp", s.address)
	} else {
		listener, err = tls.Listen("tcp", s.address, s.config)
	}
	if err != nil {
		Logger.LogFatal("Error starting TCP server.")
	}
	defer listener.Close()

	for {
		conn, _ := listener.Accept()
		client := &ServerClient{
			conn:   conn,
			Server: s,
			params: clientParams{
				canTalk: false}}
		go client.listen()
	}
}

func New(c *AgentConfig.Config, address string) *server {
	conf = c
	Logger.LogSystem(fmt.Sprintf("Init server side with address %s", address))
	server := &server{
		address: address,
		config:  nil,
	}
	return server
}

func (s *server) MasterServer() {
	if conf.Agent.Master {
		Logger.LogSystem("I'm master server")
	}
	AgentConfig.MasterVector = make(map[string]AgentConfig.MasterVectorType, 0)
	time.Sleep(time.Duration(conf.Agent.Interval) * time.Second)
}

func (c *server) insertVector() error {

	var err error

	db := Database.GetConnection()
	err = db.Ping()
	if err != nil {
		return err
	}
	for host, mapEl := range AgentConfig.Vector {
		for _, sec := range mapEl.Sectors {
			fmt.Printf("INSERT vector_stat SET host=%s,server=%s,sector=%s,status=%v,timestamp=%d\n\n",
				conf.Agent.Hostname, host, sec, mapEl.Status, time.Now().Unix())
		}
	}

	return nil
}

// func NewWithTLS(address string, certFile string, keyFile string) *server {
// 	Logger.LogInfo(fmt.Sprintf("Init tls server side with address %s", address))
// 	cert, _ := tls.LoadX509KeyPair(certFile, keyFile)
// 	config := tls.Config{
// 		Certificates: []tls.Certificate{cert},
// 	}
// 	server := &server{
// 		address: address,
// 		config:  &config,
// 	}
// 	return server
// }
