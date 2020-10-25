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

var (
	StopRequest       bool = false
	StopReply         bool = false
	masterServerIndex int  = 0
)

type ServerClient struct {
	conn          net.Conn
	masterconn    net.Conn
	configuration *AtellaConfig.Config
	neighbours    []string
	sectors       []int64
	closeRequest  chan struct{}
	CloseReply    bool
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

func (c *ServerClient) runNeighbour(addr string) {
}

// Run client
func (c *ServerClient) Run() {
	var (
		err                    error    = nil
		exit                   bool     = false
		msg                    string   = ""
		msgMap                 []string = []string{}
		status                 bool     = false
		currentNeighboursInd   int      = 0
		currentNeighboursAddr  string   = ""
		currentNeighboursAddrs []string = []string{}
		hostname               string   = ""
		vec                    AtellaConfig.VectorType
	)

	for {
		select {
		case <-c.closeRequest:
			c.configuration.Logger.LogSystem("Stopping client")
			return
		default:
		}
		if len(c.neighbours) < 1 {
			c.configuration.Logger.LogInfo("No neighbours")
			StopReply = true
		} else {
			currentNeighboursAddrs = strings.Split(c.neighbours[currentNeighboursInd],
				",")
			for _, currentNeighboursAddr = range currentNeighboursAddrs {
				if currentNeighboursInd == 0 {
					err = c.SendToMaster(fmt.Sprintf("set %s %s\n", c.configuration.Agent.Hostname,
						c.configuration.GetJsonVector()))
					if err != nil {
						c.configuration.Logger.LogError(fmt.Sprintf("%s", err))
					}
				}
				if !StopRequest {
					c.conn, err = net.DialTimeout("tcp", fmt.Sprintf("%s:5223",
						currentNeighboursAddr),
						time.Duration(c.configuration.Agent.NetTimeout)*time.Second)
					_, vectorIndex := c.configuration.GetVectorByHost(currentNeighboursAddr)
					vec = c.configuration.Vector[vectorIndex]
					if err != nil {
						c.configuration.Logger.LogWarning(fmt.Sprintf("%s", err))
						vec.Status = false
						vec.Timestamp = time.Now().Unix()
						c.configuration.Vector[vectorIndex] = vec
					} else {
						exit = false
						connbuf := bufio.NewReader(c.conn)
						err = c.Send(fmt.Sprintf("%s\n", c.configuration.Security.Code))
						if err != nil {
							status = false
							exit = true
							c.configuration.Logger.LogError(fmt.Sprintf("%s", err))
						}
						for {
							message, err := connbuf.ReadString('\n')
							if err != nil {
								if err != io.EOF {
									status = false
									exit = true
									c.configuration.Logger.LogError(fmt.Sprintf("%s", err))
								}
							}
							if exit {
								c.Close()
								vec.Status = status
								vec.Hostname = hostname
								if vectorIndex < 0 {
									c.configuration.Vector = append(c.configuration.Vector, vec)
								} else {
									vec.Timestamp = time.Now().Unix()
									c.configuration.Vector[vectorIndex] = vec
								}
								break
							}

							msg = strings.TrimRight(message, "\r\n")
							msgMap = strings.Split(msg, " ")

							c.configuration.Logger.LogInfo(fmt.Sprintf("Client receive [%s]", msg))
							switch msgMap[0] {
							case "canTalk":
								err = c.Send("hostname\n")
								if err != nil {
									status = false
									exit = true
									c.configuration.Logger.LogError(fmt.Sprintf("%s", err))
								}
							case "ackhostname":
								if len(msgMap) < 2 {
									exit = true
								} else {
									hostname = msgMap[1]
									err = c.Send(fmt.Sprintf("host %s\n", c.configuration.Agent.Hostname))
									if err != nil {
										exit = true
										c.configuration.Logger.LogError(fmt.Sprintf("%s", err))
									}
								}
							case "ackhost":
								if len(msgMap) < 2 {
									exit = true
								} else {
									if msgMap[1] == c.configuration.Agent.Hostname {
										status = true
									} else {
										status = false
									}
									err = c.Send("exit\n")
									if err != nil {
										exit = true
										c.configuration.Logger.LogError(fmt.Sprintf("%s", err))
									}
								}
							case "Bye!":
								exit = true
							}
						}
					}
				} else {
					StopReply = true
					c.configuration.Logger.LogSystem("Client ready for reload")
				}
			}
			currentNeighboursInd = (currentNeighboursInd + 1) %
				len(c.neighbours)
		}
		time.Sleep(2 * time.Second)
	}

}

// New client
func New(c *AtellaConfig.Config) *ServerClient {
	client := &ServerClient{
		CloseReply:   false,
		closeRequest: make(chan struct{})}
	client.init(c)
	return client
}

func (c *ServerClient) init(config *AtellaConfig.Config) {
	c.conn = nil
	c.configuration = config
	c.neighbours = make([]string, 0)
	c.sectors = make([]int64, 0)
	c.configuration.Vector = make([]AtellaConfig.VectorType, 0)

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

func (client *ServerClient) Reload(c *AtellaConfig.Config) {
	client.configuration.Logger.LogSystem("Client request reload")
	// Trying accuire the lock
	StopRequest = true
	// Wait until the lock is confirmed
	for !StopReply {
	}
	// Call init function for reread config
	client.init(c)
	// Release the lock
	StopReply = false
	StopRequest = false
	client.configuration.Logger.LogSystem("Client reloaded")
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
			if elExists(hosts, c.configuration.Agent.Hostname) {
				// Saving index of sector into array
				sector = append(sector, int64(i))
				c.configuration.Logger.LogInfo(fmt.Sprintf("Sector: %s [%d]",
					c.configuration.Sectors[i].Sector, i))
				// Loop for seach and adding neighbours in my sectors
				for l := 1; int64(l) <= c.configuration.Agent.HostCnt; l = l + 1 {
					hosts_next := strings.Split(c.configuration.Sectors[i].Config.Hosts[(j+l)%
						hostsCnt], " ")
					hosts_prev := strings.Split(
						c.configuration.Sectors[i].Config.Hosts[(j-l+hostsCnt)%hostsCnt], " ")
					// if next host is not me
					if !elExists(hosts_next, c.configuration.Agent.Hostname) {
						c.AddHost(hosts_next[0], c.configuration.Sectors[i].Sector)
					}
					// if prev host is not me
					if !elExists(hosts_prev, c.configuration.Agent.Hostname) {
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
				Timestamp: -1,
				Sectors:   make([]string, 0)}
		} else {
			vec = c.configuration.Vector[index]
		}
		// Save time of change
		vec.Timestamp = time.Now().Unix()

		// If sectors array doesn.t include host sector - append him into list
		if !elExists(vec.Sectors, sector) {
			vec.Sectors = append(vec.Sectors,
				sector)
			c.configuration.Logger.LogInfo(fmt.Sprintf("Added sector [%s] for host [%s]",
				sector, h))
		}

		// If a neighbour doesn.t added, adding host
		if !elExists(c.neighbours, h) {
			c.neighbours = append(c.neighbours, h)
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

// Function check array and return true if item exist
func elExists(array []string, item string) bool {
	for i := 0; i < len(array); i = i + 1 {
		if array[i] == item {
			return true
		}
	}
	return false
}

// Function send vector to one of master servers
func (c *ServerClient) SendToMaster(query string) error {
	var err error = nil

	// If i am a master server, save client vector to local master vector
	if c.configuration.Agent.Master {
		var vec []AtellaConfig.VectorType
		json.Unmarshal(c.configuration.GetJsonVector(), &vec)
		c.configuration.MasterVector[c.configuration.Agent.Hostname] = vec
		return nil
	}

	// Exit if we don.t have master servers
	if c.configuration.CurrentMasterServerIndex < 0 {
		return fmt.Errorf("Master servers not specifiyed")
	}

	// Loop because link to current master server may be broken
	for {
		masterAddr := strings.Split(
			c.configuration.MasterServers.Hosts[c.configuration.CurrentMasterServerIndex], " ")
		c.masterconn, err = net.DialTimeout("tcp", fmt.Sprintf("%s:5223",
			masterAddr[0]), time.Duration(c.configuration.Agent.NetTimeout)*time.Second)
		// If connection have any of errors - try next server
		if err != nil {
			c.configuration.CurrentMasterServerIndex =
				c.configuration.CurrentMasterServerIndex + 1
			c.configuration.CurrentMasterServerIndex =
				c.configuration.CurrentMasterServerIndex %
					len(c.configuration.MasterServers.Hosts)
		} else {
			// If connection is ok, send vector
			_, err = c.masterconn.Write(
				[]byte(fmt.Sprintf("%s\n", c.configuration.Security.Code)))
			_, err = c.masterconn.Write([]byte(query))
			c.masterconn.Close()
			masterServerIndex = c.configuration.CurrentMasterServerIndex
			break
		}

		// If we try all servers and all servers unreacheble - return error
		if c.configuration.CurrentMasterServerIndex == masterServerIndex {
			return fmt.Errorf("Could not connect to any of masters")
		}
	}

	return err
}

// Function for stopping client
func (c *ServerClient) Stop() {
	close(c.closeRequest)
}
