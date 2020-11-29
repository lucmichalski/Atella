package AtellaClient

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strings"
	"time"

	"../AtellaConfig"
)

const (
	okMsg  string = "+OK"
	errMsg string = "-ERR"
)

var (
	masterServerIndex int = 0
)

type ServerClient struct {
	master        master
	neighbours    []neigbour
	configuration *AtellaConfig.Config
	stopRequest   chan struct{}
	sectors       []int64
}

type neigbour struct {
	conn            net.Conn
	connError       bool
	emptyMessageCnt uint64
	stopReply       bool
	address         string
	port            int16
}

type master struct {
	conn      net.Conn
	connError bool
	stopReply bool
}

// Function send string via connection
func (c *neigbour) Send(message string) error {
	_, err := c.conn.Write([]byte(message))
	return err
}

// Function send byte array via connection
func (c *neigbour) SendBytes(b []byte) error {
	_, err := c.conn.Write(b)
	return err
}

// Function return connection descriptor
func (c *neigbour) Conn() net.Conn {
	return c.conn
}

// Function close the connection
func (c *neigbour) Close() error {
	return c.conn.Close()
}

func (client *ServerClient) runNeighbour(c *neigbour) error {
	var (
		err      error = nil
		fin      bool  = false
		exit     bool  = false
		vec      AtellaConfig.VectorType
		status   bool          = false
		msg      string        = ""
		hostname string        = "unknown"
		msgMap   []string      = []string{}
		connbuf  *bufio.Reader = nil
	)

	client.configuration.Logger.LogInfo(fmt.Sprintf("[Client] Start routine for %s:%d",
		c.address, c.port))

	_, vectorIndex := client.configuration.GetVectorByHost(c.address)
	if vectorIndex < 0 {
		client.configuration.Logger.LogError(fmt.Sprintf("[Client] %s", err))
		return fmt.Errorf("Host [%s] are not present in vector array!", c.address)
	}

	go func() {
		<-client.stopRequest
		exit = true
		c.connError = true
		if c.conn != nil {
			c.conn.Close()
		}
	}()

	// Infinity loop for requests
	for {
		AtellaConfig.Pause(client.configuration.Agent.Interval, &exit)

		// If connection has error - reopen connection
		if c.connError {
			if exit {
				break
			}
			c.conn, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%d",
				c.address, c.port),
				time.Duration(client.configuration.Agent.NetTimeout)*time.Second)
			// if connection failed print error
			if err != nil {
				client.configuration.Logger.LogError(fmt.Sprintf("[Client] %s", err))
				c.connError = true
			} else {
				c.connError = false
				connbuf = bufio.NewReader(c.conn)
			}
			continue
		}

		vec = client.configuration.Vector[vectorIndex]
		fin = false
		err = c.Send(fmt.Sprintf("auth %s\n", client.configuration.Security.Code))
		if err != nil {
			status = false
			c.connError = true
			client.configuration.Logger.LogError(
				fmt.Sprintf("[Client] neighbour [%s]. Security - %s", c.address, err))
			continue
		}

		// auth - ack - hostname - ack - host - ack
		for {
			if fin {
				vec.Status = status
				vec.Hostname = hostname
				vec.Timestamp = time.Now().Unix()
				client.configuration.Vector[vectorIndex] = vec
				break
			}

			message, err := connbuf.ReadString('\n')
			if err != nil && err != io.EOF {
				fin = true
				status = false
				c.connError = true
				vec.Status = status
				client.configuration.Vector[vectorIndex] = vec
				client.configuration.Logger.LogError(
					fmt.Sprintf("[Client] Neighbour [%s]. Read - %s", c.address, err))
				continue
			}

			msg = strings.TrimRight(message, "\r\n")
			msgMap = strings.Split(msg, " ")
			client.configuration.Logger.LogInfo(
				fmt.Sprintf("[Client] Neighbour [%s]. Receive [%s]", c.address, msg))

			if msg == "" {
				if c.emptyMessageCnt > 5 {
					client.configuration.Logger.LogWarning(
						fmt.Sprintf("[Client] Received %d empty messages. Force closing connection.",
							c.emptyMessageCnt))
					c.connError = true
					fin = true
					continue
				}
				c.emptyMessageCnt = c.emptyMessageCnt + 1
			} else {
				c.emptyMessageCnt = 0
			}

			if msgMap[0] == errMsg {
				client.configuration.Logger.LogError(
					fmt.Sprintf("[Client] Neighbour [%s]. Receive %s", c.address, errMsg))
				continue
			} else if msgMap[0] != okMsg {
				client.configuration.Logger.LogError(
					fmt.Sprintf("[Client] Neighbour [%s]. Receive !%s", c.address, okMsg))
				continue
			}

			switch msgMap[1] {
			case "ack":
				switch msgMap[2] {
				case "auth":
					err = c.Send("get hostname\n")
					if err != nil {
						status = false
						fin = true
						c.connError = true
						client.configuration.Logger.LogError(
							fmt.Sprintf("[Client] Neighbour [%s]. Send hostname - %s",
								c.address, err))
					}
				case "hostname":
					if len(msgMap) < 4 {
						fin = true
						status = false
						client.configuration.Logger.LogError(
							fmt.Sprintf("[Client] Neighbour [%s]. Msg len expected ack < 4",
								c.address))
					}
					hostname = msgMap[3]
					err = c.Send(fmt.Sprintf("set host %s\n", client.configuration.Agent.Hostname))
					if err != nil {
						fin = true
						status = false
						c.connError = true
						client.configuration.Logger.LogError(
							fmt.Sprintf("[Client] Neighbour [%s]. Send host - %s",
								c.address, err))
					}
				case "host":
					if len(msgMap) < 4 {
						fin = true
						status = false
						client.configuration.Logger.LogError(
							fmt.Sprintf("[Client] Neighbour [%s]. Msg len expected ack < 4",
								c.address))
					}
					if msgMap[3] == client.configuration.Agent.Hostname {
						status = true
					} else {
						status = false
					}
					vec.Status = status
					vec.Hostname = hostname
					vec.Timestamp = time.Now().Unix()
					client.configuration.Vector[vectorIndex] = vec
				}

			}
		}
	}
	c.stopReply = true
	client.configuration.Logger.LogSystem(
		fmt.Sprintf("Routine for %s:%d stopped",
			c.address, c.port))
	return nil
}

// Run client
func (c *ServerClient) Run() {
	// Init neighbours goroutines
	for n := 0; n < len(c.neighbours); n = n + 1 {
		go c.runNeighbour(&c.neighbours[n])
	}
	go c.runMasterClient()
}

// New client
func New(c *AtellaConfig.Config) *ServerClient {
	client := &ServerClient{}
	client.init(c)
	return client
}

func (c *ServerClient) init(configuration *AtellaConfig.Config) {
	c.neighbours = make([]neigbour, 0)
	c.sectors = make([]int64, 0)
	c.configuration = configuration
	c.configuration.Vector = make([]AtellaConfig.VectorType, 0)
	c.stopRequest = make(chan struct{})

	// Selecting pseudo-random master from config
	if len(c.configuration.MasterServers.Hosts) < 1 {
		c.configuration.CurrentMasterServerIndex = -1
		c.configuration.Logger.LogWarning(fmt.Sprintf("Master servers not specifiyed!"))
	} else if !c.configuration.Agent.Master {
		masterServerIndex = rand.Int() % len(c.configuration.MasterServers.Hosts)
		c.configuration.CurrentMasterServerIndex = 0
		c.configuration.Logger.LogSystem(fmt.Sprintf("Use [%s] as master server",
			c.configuration.MasterServers.Hosts[masterServerIndex]))
	}

	c.GetMySector()
	c.configuration.Logger.LogSystem("Init client side")
}

// Function find and save sector indexes
func (c *ServerClient) GetMySector() {
	var (
		sector     []int64 = []int64{}
		sectorsCnt         = len(c.configuration.Sectors)
	)
	for i := 0; i < sectorsCnt; i = i + 1 {
		hostsCnt := len(c.configuration.Sectors[i].Config.Hosts)
		for j := 0; j < hostsCnt; j = j + 1 {
			hosts := strings.Split(c.configuration.Sectors[i].Config.Hosts[j], " ")
			// If current host equal my hostname
			if stringElExists(hosts, c.configuration.Agent.Hostname) {
				// Saving index of sector into array
				if !int64ElExists(sector, int64(i)) {
					sector = append(sector, int64(i))
					c.configuration.Logger.LogInfo(fmt.Sprintf("Added sector for my host: %s [Index %d]",
						c.configuration.Sectors[i].Sector, i))
				}
				// Loop for seach and adding neighbours in my sectors
				for l := 1; int64(l) <= c.configuration.Agent.HostCnt; l = l + 1 {
					hosts_next := strings.Split(c.configuration.Sectors[i].Config.Hosts[(j+l)%
						hostsCnt], " ")
					hosts_prev := strings.Split(
						c.configuration.Sectors[i].Config.Hosts[(j-l+hostsCnt)%hostsCnt], " ")
					// if next host is not me
					if !stringElExists(hosts_next, c.configuration.Agent.Hostname) {
						c.AddHost(hosts_next[0], c.configuration.Sectors[i].Sector)
					}
					// if prev host is not me
					if !stringElExists(hosts_prev, c.configuration.Agent.Hostname) {
						c.AddHost(hosts_prev[0], c.configuration.Sectors[i].Sector)
					}
				}
			}
		}
	}
	c.sectors = sector
}

// Function add non-existing host in vector and neighbours array
func (c *ServerClient) AddHost(host string, sector string) {
	var vec AtellaConfig.VectorType
	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		// Getting vector index for current host
		_, index := c.configuration.GetVectorByHost(h)

		// If host doesn.t have a vector - create new, else use existing
		if index < 0 {
			vec = AtellaConfig.VectorType{
				Host:      h,
				Hostname:  "unknown",
				Status:    false,
				Interval:  c.configuration.Agent.Interval,
				Timestamp: 0,
				Sectors:   make([]string, 0)}
		} else {
			vec = c.configuration.Vector[index]
		}
		// Save time of change
		vec.Timestamp = time.Now().Unix()

		// If sectors array doesn.t include host sector - append him into list
		if !stringElExists(vec.Sectors, sector) {
			vec.Sectors = append(vec.Sectors,
				sector)
			c.configuration.Logger.LogInfo(fmt.Sprintf("Added sector [%s] for host [%s]",
				sector, h))
		}

		// If a neighbour doesn.t added, adding host
		if !neighbourElExistsByAddress(c.neighbours, h) {
			n := neigbour{
				conn:            nil,
				connError:       true,
				address:         h,
				port:            5223,
				emptyMessageCnt: 0}
			c.neighbours = append(c.neighbours, n)
			c.configuration.Logger.LogInfo(fmt.Sprintf("Added a neighbour host [%s]",
				h))
		}

		// If the vector did not exist, saving, else - override existing
		if index < 0 {
			c.configuration.Vector = append(c.configuration.Vector, vec)
		} else {
			c.configuration.Vector[index] = vec
		}
	}
}

// Function check string array and return true if item exist
func stringElExists(array []string, item string) bool {
	for i := 0; i < len(array); i = i + 1 {
		if array[i] == item {
			return true
		}
	}
	return false
}

// Function check string array and return true if item exist
func int64ElExists(array []int64, item int64) bool {
	for i := 0; i < len(array); i = i + 1 {
		if array[i] == item {
			return true
		}
	}
	return false
}

// Function check string array and return true if item exist
func neighbourElExistsByAddress(array []neigbour, addr string) bool {
	for i := 0; i < len(array); i = i + 1 {
		if array[i].address == addr {
			return true
		}
	}
	return false
}

// Function starts and handle connection to master server
func (c *ServerClient) runMasterClient() error {
	var (
		err        error = nil
		masterAddr []string
		exit       bool = false
	)
	c.master.connError = true

	// Exit if we don.t have master servers
	if c.configuration.CurrentMasterServerIndex < 0 {
		return fmt.Errorf("Master servers not specifiyed")
	}

	go func() {
		<-c.stopRequest
		exit = true
		c.master.connError = true
		if c.master.conn != nil {
			c.master.conn.Close()
		}
	}()

	// Loop because link to current master server may be broken
	for !exit {
		AtellaConfig.Pause(c.configuration.Agent.Interval, &exit)

		// If i am a master server, save client vector to local master vector
		if c.configuration.Agent.Master {
			var vec []AtellaConfig.VectorType
			json.Unmarshal(c.configuration.GetJsonVector(), &vec)
			c.configuration.MasterVector[c.configuration.Agent.Hostname] = vec
			continue
		}

		// If connection has error - reopen connection
		if c.master.connError {
			if exit {
				break
			}
			for !exit {
				masterAddr = strings.Split(
					c.configuration.MasterServers.Hosts[c.configuration.CurrentMasterServerIndex], " ")
				c.master.conn, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%d",
					masterAddr[0], 5223),
					time.Duration(c.configuration.Agent.NetTimeout)*time.Second)
				// if connection failed print error
				if err != nil {
					c.configuration.Logger.LogError(fmt.Sprintf("%s", err))
					// If connection have any of errors - try next server
					c.configuration.CurrentMasterServerIndex =
						c.configuration.CurrentMasterServerIndex + 1
					c.configuration.CurrentMasterServerIndex =
						c.configuration.CurrentMasterServerIndex %
							len(c.configuration.MasterServers.Hosts)
					c.master.connError = true

					// If we try all servers and all servers unreacheble - return error
					if c.configuration.CurrentMasterServerIndex == masterServerIndex {
						c.configuration.Logger.LogError("Could not connect to any of masters")
						// 	return fmt.Errorf("Could not connect to any of masters")
					}
				} else {
					c.master.connError = false
					masterServerIndex = c.configuration.CurrentMasterServerIndex
					break
				}
			}
		}

		// If connection is ok, send vector
		c.sendVectorToMaster(
			[]byte(fmt.Sprintf("set vector %s %s\n", c.configuration.Agent.Hostname,
				c.configuration.GetJsonVector())))
	}

	c.master.stopReply = true
	c.configuration.Logger.LogSystem(fmt.Sprintf("Master client connection stoped"))
	return nil
}

// Function send vector to one of master servers
func (c *ServerClient) sendVectorToMaster(query []byte) error {
	var err error = nil

	_, err = c.master.conn.Write(
		[]byte(fmt.Sprintf("auth %s\n", c.configuration.Security.Code)))

	if err != nil {
		c.master.connError = true
		c.configuration.Logger.LogError(
			fmt.Sprintf("[Client] master connection [security]. Error - %s",
				err))
		return err
	}

	_, err = c.master.conn.Write(query)
	if err != nil {
		c.master.connError = true
		c.configuration.Logger.LogError(
			fmt.Sprintf("[Client] master connection [query]. Error - %s",
				err))
		return err
	}

	return err
}

func (client *ServerClient) Reload(c *AtellaConfig.Config) {
	client.configuration.Logger.LogSystem("[Client] Reloading client")

	// Trying accuire the lock
	close(client.stopRequest)
	// client.master.stopRequest = true
	// for i := 0; i < len(client.neighbours); i = i + 1 {
	// 	client.neighbours[i].stopRequest = true
	// }

	// Wait until the lock is confirmed
	confirm := false
	for !confirm {
		confirm = true
		for i := 0; i < len(client.neighbours); i = i + 1 {
			if !client.neighbours[i].stopReply {
				confirm = false
			}
		}
		if !client.master.stopReply {
			confirm = false
		}
	}

	// Call init function for reread config
	client.init(c)
	// Release the lock
	// client.master.stopRequest = false
	// for i := 0; i < len(client.neighbours); i = i + 1 {
	// 	client.neighbours[i].stopRequest = false
	// }
	client.Run()
	client.configuration.Logger.LogSystem("[Client] Client reloaded")
}

// Function for stopping client
func (client *ServerClient) Stop() {
	client.configuration.Logger.LogSystem("[Client] Stopping client")

	// Trying accuire the lock
	close(client.stopRequest)
	// client.master.stopRequest = true
	// for i := 0; i < len(client.neighbours); i = i + 1 {
	// 	client.neighbours[i].stopRequest = true
	// }

	// Wait until the lock is confirmed
	confirm := false
	for !confirm {
		confirm = true
		for i := 0; i < len(client.neighbours); i = i + 1 {
			if !client.neighbours[i].stopReply {
				confirm = false
			}
		}
		if !client.master.stopReply {
			confirm = false
		}
	}

	client.configuration.Logger.LogSystem("[Client] Client stopped")
}
