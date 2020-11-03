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
	"../AtellaDatabase"
)

// Parameters for each client
type clientParams struct {
	canTalk                 bool
	id                      uint64
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
	closeRequest     chan struct{}
	CloseReplyServer bool
	CloseReplyMaster bool
}

// Processing client
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
	s.configuration.Logger.LogInfo(fmt.Sprintf("New connect from %s[%d], can talk with him - %t",
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
	s.configuration.Logger.LogInfo(fmt.Sprintf("Client [%d] go away", c.params.id))
}

// Processing each message, receiving from clients
func (s *AtellaServer) OnNewMessage(c *ServerClient, message string) bool {
	var (
		msg    = strings.TrimRight(message, "\r\n")
		msgMap = strings.Split(msg, " ")
	)

	// Commands, dont.t require security check
	switch msgMap[0] {
	case "quit", "exit":
		c.Send(fmt.Sprintf("Bye!\n"))
		return true
	case "export":
		if len(msgMap) > 1 {
			if msgMap[1] == "vector" {
				c.Send(fmt.Sprintf("ackvector %s\n", s.configuration.GetJsonVector()))
				s.configuration.PrintJsonVector()
			} else if msgMap[1] == "master" {
				c.Send(fmt.Sprintf("ackmaster %s\n", s.configuration.GetJsonMasterVector()))
				s.configuration.PrintJsonMasterVector()
			}
		}
	case "ping":
		c.Send("pong")
	}

	// Commands, require security check
	if c.params.canTalk {
		s.configuration.Logger.LogInfo(fmt.Sprintf("Server receive [%s]", msg))
		switch msgMap[0] {
		case "who":
			c.Send(fmt.Sprintf("Id: %d\n", c.params.id))
		case "host":
			if len(msgMap) > 1 {
				c.params.currentClientHostname = msgMap[1]
				c.Send(fmt.Sprintf("ackhost %s\n", c.params.currentClientHostname))
			}
		case "hostname":
			c.Send(fmt.Sprintf("ackhostname %s\n", s.configuration.Agent.Hostname))
		case "version":
			c.Send(fmt.Sprintf("ackversion %s\n", AtellaConfig.Version))
		case "help":
			c.help()
		// case "update":
		// 	if len(msgMap) > 1 {
		// 		version := msgMap[1]
		// 		AtellaLogger.LogSystem(fmt.Sprintf("Receive update to %s from %s",
		// 			version, AtellaConfig.Version))
		// 		if version != AtellaConfig.Version {
		// 			AtellaLogger.LogSystem(fmt.Sprintf("Initiate install %s",
		// 				version))
		// 			cmd := exec.Command(fmt.Sprintf("%s/atella-cli",
		// 				AtellaConfig.BinPrefix),
		// 				"-cmd", "update", "-to-version", version)
		// 			err := cmd.Start()
		// 			// err := syscall.Exec(fmt.Sprintf("%s/atella-cli",
		// 			// 	AtellaConfig.BinPrefix),
		// 			// 	[]string{fmt.Sprintf("%s/atella-cli",
		// 			// 		AtellaConfig.BinPrefix),
		// 			// 		"-cmd", "update", "-to-version", version}, os.Environ())

		// 			if err != nil {
		// 				AtellaLogger.LogError("Failed exec cli for update")
		// 				AtellaLogger.LogError(
		// 					fmt.Sprintf("%s/atella-cli -cmd update -to-version %s",
		// 						AtellaConfig.BinPrefix, version))
		// 				AtellaLogger.LogError(fmt.Sprintf("%s", err))
		// 			}
		// 			cmd.Process.Release()
		// 		}
		// 	}
		case "set":
			if len(msgMap) > 2 {
				c.params.currentClientHostname = msgMap[1]
				c.params.currentClientVectorJson = msgMap[2]
				if c.params.currentClientHostname != "" &&
					c.params.currentClientVectorJson != "" {
					var vec []AtellaConfig.VectorType
					json.Unmarshal([]byte(c.params.currentClientVectorJson), &vec)
					s.configuration.MasterVectorMutex.Lock()
					s.configuration.MasterVector[c.params.currentClientHostname] = vec
					s.configuration.MasterVectorMutex.Unlock()
				}
			}
		default:
			s.configuration.Logger.LogWarning(fmt.Sprintf("Unknown cmd %s [%s]\n",
				msgMap[0], msg))
		}
	} else if msg == s.configuration.Security.Code {
		s.configuration.Logger.LogInfo(fmt.Sprintf("Server receive [%s], set canTalk -> true",
			msg))
		c.Send("canTalk\n")
		c.params.canTalk = true
	} else {
		s.configuration.Logger.LogInfo(fmt.Sprintf("Server receive [%s], can't talk - ignore",
			msg))
	}
	return false
}

func (c *ServerClient) help() {
	c.Send("ping\n")
	c.Send("export {vector/master}\n")
	c.Send("host {hostname}\n")
	c.Send("hostname\n")
	c.Send("version\n")
	c.Send("set {hostname} {vector}\n")
	c.Send("exit\n")
}

// Listen for connections
func (s *AtellaServer) Listen() {
	var listener *net.TCPListener
	var err error
	address, err := net.ResolveTCPAddr("tcp", s.address)
	if s.tlsConfig == nil {
		listener, err = net.ListenTCP("tcp", address)
	} else {
		s.configuration.Logger.LogFatal("Tls server are not implemented")
		// listener, err = tls.ListenTCP("tcp", address, s.tlsConfig)
	}
	if err != nil {
		s.configuration.Logger.LogFatal(fmt.Sprintf("Error starting TCP server. %s", err))
	}
	defer listener.Close()

	for {
		select {
		case <-s.closeRequest:
			s.configuration.Logger.LogSystem("Stopping server")
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
			s.configuration.Logger.LogInfo(
				fmt.Sprintf("Failed to accept connection: %s", err.Error()))
			continue
		}
		client := &ServerClient{
			conn:   conn,
			Server: s,
			params: clientParams{
				canTalk: false}}
		go client.listen()
	}
}

// Create new server
func New(c *AtellaConfig.Config, address string) *AtellaServer {
	c.Logger.LogSystem(fmt.Sprintf("Init server side with address %s",
		address))
	server := &AtellaServer{
		address:          address,
		tlsConfig:        nil,
		configuration:    c,
		closeRequest:     make(chan struct{}),
		CloseReplyServer: false,
		CloseReplyMaster: false}
	return server
}

// Function impement master server logic
func (s *AtellaServer) MasterServer() {
	if s.configuration.Agent.Master {
		s.configuration.Logger.LogSystem("I'm master server")
	} else {
		s.configuration.Logger.LogSystem("I'm not a master server")
		s.CloseReplyMaster = true
		return
	}

	s.configuration.MasterVectorMutex.Lock()
	s.configuration.MasterVector = make(map[string][]AtellaConfig.VectorType, 0)
	s.configuration.MasterVectorMutex.Unlock()
	for {
		select {
		case <-s.closeRequest:
			s.configuration.Logger.LogSystem("Stopping master server")
			s.CloseReplyMaster = true
			return
		default:
		}
		time.Sleep(time.Duration(s.configuration.Agent.Interval) * time.Second)
		s.insertVector()
	}
}

// Function for stopping server
func (s *AtellaServer) Stop() {
	close(s.closeRequest)
}

func (s *AtellaServer) insertVector() error {

	count, _ := AtellaDatabase.SelectQuery(fmt.Sprintf(
		"SELECT * FROM vector WHERE master='%s'",
		s.configuration.Agent.Hostname))
	if count > 0 {

	}
	return nil
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
