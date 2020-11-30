package AtellaServer

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	"../AtellaConfig"
)

const (
	okMsg  string = "+OK"
	errMsg string = "-ERR"
)

// Parameters for each client
type clientParams struct {
	canTalk                 bool
	id                      uint64
	emptyMessageCnt         uint64
	currentClientHostname   string
	currentClientVectorJson string
}

// Client description
type ServerClient struct {
	conn   net.Conn
	Server *AtellaServer
	params clientParams
}

// Server parameters
type AtellaServer struct {
	address          string
	configuration    *AtellaConfig.Config
	global           uint64
	tlsConfig        *tls.Config
	reloadRequest    chan struct{}
	stopRequest      chan struct{}
	CloseReplyServer bool
	CloseReplyMaster bool
}

// Processing client
func (c *ServerClient) listen() {
	c.Server.OnNewClient(c)
	reader := bufio.NewReader(c.conn)
	var exit = false

	go func() {
		<-c.Server.stopRequest
		exit = true
		if c.conn != nil {
			c.conn.Close()
		}
	}()

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

// Function send string via connection
func (c *ServerClient) Send(message string) error {
	_, err := c.conn.Write([]byte(message))
	return err
}

// Function send byte array via connection
func (c *ServerClient) SendBytes(b []byte) error {
	_, err := c.conn.Write(b)
	return err
}

// Function return connection descriptor
func (c *ServerClient) Conn() net.Conn {
	return c.conn
}

// Function close the connection
func (c *ServerClient) Close() error {
	return c.conn.Close()
}

// Processing client connection
func (s *AtellaServer) OnNewClient(c *ServerClient) {
	s.configuration.Logger.LogInfo(fmt.Sprintf("[Server] New connect from %s[%d], can talk with him - %t",
		c.conn.RemoteAddr(), s.global, c.params.canTalk))
	// Logical splitting clients by pseudo-unique id
	c.params.id = s.global
	s.global = s.global + 1
	if s.global == math.MaxInt64 {
		s.global = 0
	}
	c.params.currentClientHostname = ""
	c.params.currentClientVectorJson = ""
}

// Processing client disconnection
func (s *AtellaServer) OnClientConnectionClosed(c *ServerClient, err error) {
	s.configuration.Logger.LogInfo(fmt.Sprintf("[Server] Client [%d] go away", c.params.id))
}

// Processing each message, receiving from clients
func (s *AtellaServer) OnNewMessage(c *ServerClient, message string) bool {
	var (
		msg    = strings.TrimRight(message, "\r\n")
		msgMap = strings.Split(msg, " ")
	)
	if msg == "" {
		if c.params.emptyMessageCnt > 5 {
			s.configuration.Logger.LogWarning(
				fmt.Sprintf("[Server] Server receive %d empty messages. Force closing connection.",
					c.params.emptyMessageCnt))
			return true
		}
		c.params.emptyMessageCnt = c.params.emptyMessageCnt + 1
	} else {
		c.params.emptyMessageCnt = 0
	}

	s.configuration.Logger.LogInfo(fmt.Sprintf("[Server] Server receive [%s | %d]", msg, len(msg)))
	switch msgMap[0] {
	// Commands, dont.t require security check
	case "quit", "exit":
		c.Send(fmt.Sprintf("%s bye!\n", okMsg))
		return true

	case "export":
		if len(msgMap) > 1 {
			if msgMap[1] == "vector" {
				c.Send(fmt.Sprintf("%s ack vector %s\n", okMsg, s.configuration.GetJsonVector()))
				s.configuration.PrintJsonVector()
			} else if msgMap[1] == "master" {
				c.Send(fmt.Sprintf("%s ack master %s\n", okMsg, s.configuration.GetJsonMasterVector()))
				s.configuration.PrintJsonMasterVector()
			}
		}

	case "ping":
		c.Send("pong")

	// Commands, require security check
	case "get":
		switch msgMap[1] {
		case "whoami":
			c.Send(fmt.Sprintf("%s ack whoami %d\n", okMsg, c.params.id))
		case "hostname":
			c.Send(fmt.Sprintf("%s ack hostname %s\n", okMsg, s.configuration.Agent.Hostname))
		case "version":
			c.Send(fmt.Sprintf("%s ack version %s\n", okMsg, AtellaConfig.Version))
		default:
			s.configuration.Logger.LogWarning(fmt.Sprintf("[Server] Unknown cmd %s [%s]\n",
				msgMap[1], msg))
		}

	case "set":
		switch msgMap[1] {
		case "host":
			if len(msgMap) > 2 {
				c.params.currentClientHostname = msgMap[2]
				c.Send(fmt.Sprintf("%s ack host %s\n", okMsg, c.params.currentClientHostname))
			} else {
				c.Send(fmt.Sprintf("%s set host\n", errMsg))
			}
		case "vector":
			if len(msgMap) > 3 {
				c.params.currentClientHostname = msgMap[2]
				c.params.currentClientVectorJson = msgMap[3]
				var vec []AtellaConfig.VectorType
				json.Unmarshal([]byte(c.params.currentClientVectorJson), &vec)
				s.configuration.MasterVectorSetElement(c.params.currentClientHostname, vec)
				c.Send(fmt.Sprintf("%s ack set\n", okMsg))
			} else {
				c.Send(fmt.Sprintf("%s set vector\n", errMsg))
			}
		default:
			s.configuration.Logger.LogWarning(fmt.Sprintf("[Server] Unknown cmd %s [%s]\n",
				msgMap[1], msg))
		}

	case "help":
		c.help()

	// Auth command
	case "auth":
		if len(msgMap) > 1 && msgMap[1] == s.configuration.Security.Code {
			s.configuration.Logger.LogInfo(fmt.Sprintf("[Server] Code accept, auth success"))
			c.Send(fmt.Sprintf("%s ack auth\n", okMsg))
			c.params.canTalk = true
		} else {
			s.configuration.Logger.LogInfo(fmt.Sprintf("[Server] Server receive [%s], failed auth",
				msgMap[1]))
			c.Send(fmt.Sprintf("%s auth\n", errMsg))

		}
	default:
		s.configuration.Logger.LogWarning(fmt.Sprintf("[Server] Unknown cmd %s [%s]\n",
			msgMap[0], msg))
	}

	return false
}

func (c *ServerClient) help() {
	c.Send("ping\n")
	c.Send("auth {code}\n")
	c.Send("export {vector/master}\n")
	c.Send("get whoami\n")
	c.Send("get hostname\n")
	c.Send("get version\n")
	c.Send("set host {hostname}\n")
	c.Send("set vector {hostname} {vector}\n")
	c.Send("exit\n")
	c.Send(fmt.Sprintf("%s\n", okMsg))
}

// Listen for connections
func (s *AtellaServer) Listen() {
	var listener *net.TCPListener
	var err error
	address, err := net.ResolveTCPAddr("tcp", s.address)
	if s.tlsConfig == nil {
		listener, err = net.ListenTCP("tcp", address)
	} else {
		s.configuration.Logger.LogFatal("[Server] Tls server are not implemented")
		// listener, err = tls.ListenTCP("tcp", address, s.tlsConfig)
	}
	if err != nil {
		s.configuration.Logger.LogFatal(fmt.Sprintf("[Server] Error starting TCP server. %s", err))
	}
	defer listener.Close()

	for {
		select {
		case <-s.stopRequest:
			s.configuration.Logger.LogSystem("[Server] Stopping server")
			s.CloseReplyServer = true
			return
		default:
		}
		listener.SetDeadline(time.Now().Add(1e9))
		conn, err := listener.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}
			s.configuration.Logger.LogError(
				fmt.Sprintf("[Server] Failed to accept connection: %s", err.Error()))
			continue
		}
		client := &ServerClient{
			conn:   conn,
			Server: s,
			params: clientParams{
				canTalk:         false,
				emptyMessageCnt: 0}}
		go client.listen()
	}
}

// Create new server
func New(c *AtellaConfig.Config, address string) *AtellaServer {
	c.Logger.LogSystem(fmt.Sprintf("[Server] Init server side with address %s",
		address))
	server := &AtellaServer{
		address:          address,
		tlsConfig:        nil,
		configuration:    c,
		stopRequest:      make(chan struct{}),
		reloadRequest:    make(chan struct{}),
		CloseReplyServer: false,
		CloseReplyMaster: false}
	return server
}

// Function for stopping server
func (s *AtellaServer) Stop() {
	close(s.stopRequest)
	for !s.CloseReplyServer || !s.CloseReplyMaster {
	}
	s.configuration.Logger.LogSystem("[Server] Server stopped")
}

// func NewWithTLS(address string, certFile string, keyFile string) *AtellaServer {
// 	AtellaLogger.LogInfo(fmt.Sprintf("Init tls server side with address %s", address))
// 	cert, _ := tls.LoadX509KeyPair(certFile, keyFile)
// 	config := tls.Config{
// 		Certificates: []tls.Certificate{cert},
// 	}
// 	server := &AtellaServer{
// 		address: address,
// 		config:  &config,
// 	}
// 	return server
// }
