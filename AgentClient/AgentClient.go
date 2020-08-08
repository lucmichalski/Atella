package AgentClient

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"../AgentConfig"
	"../Logger"
)

var (
	StopRequest              bool = false
	StopReply                bool = false
	masterServerIndex        int  = 0
	currentMasterServerIndex int  = 0
)

type ServerClient struct {
	conn       net.Conn
	masterconn net.Conn
	conf       *AgentConfig.Config
	neighbours []string
	sectors    []int64
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

// Run client
func (c *ServerClient) Run() {
	var (
		err                   error    = nil
		exit                  bool     = false
		msg                   string   = ""
		msg_map               []string = []string{}
		status                bool     = false
		currentNeighboursInd  int      = 0
		currentNeighboursAddr string   = ""
		vec                   AgentConfig.VectorType
	)

	for {
		if len(c.neighbours) < 1 {
			Logger.LogInfo("No neighbours")
			StopReply = true
		} else {
			currentNeighboursAddr = c.neighbours[currentNeighboursInd]
			if currentNeighboursInd == 0 {
				err = c.sendVector()
				if err != nil {
					Logger.LogError(fmt.Sprintf("%s", err))
				}
			}
			if !StopRequest {
				c.conn, err = net.Dial("tcp", fmt.Sprintf("%s:5223",
					currentNeighboursAddr))
				vectorIndex := getVectorIndexByHost(currentNeighboursAddr)
				vec.Host = currentNeighboursAddr
				if err != nil {
					Logger.LogWarning(fmt.Sprintf("%s", err))
					vec.Status = false
				} else {
					exit = false
					connbuf := bufio.NewReader(c.conn)
					for {
						message, _ := connbuf.ReadString('\n')
						msg = strings.TrimRight(message, "\r\n")
						msg_map = strings.Split(msg, " ")

						Logger.LogInfo(fmt.Sprintf("Client receive [%s]", msg))
						switch msg_map[0] {
						case "Meow?":
							err = c.Send("Meow!\n")
							if err != nil {
								status = false
								exit = true
								Logger.LogError(fmt.Sprintf("%s", err))
							}
						case "canTalk":
							err = c.Send(fmt.Sprintf("host %s\n", c.conf.Agent.Hostname))
							if err != nil {
								status = false
								exit = true
								Logger.LogError(fmt.Sprintf("%s", err))
							}
						case "ackhost":
							status = true
							err = c.Send("exit\n")
							if err != nil {
								exit = true
								Logger.LogError(fmt.Sprintf("%s", err))
							}
						case "Bye!":
							exit = true
						}
						if exit {
							c.Close()
							vec.Status = status
							if vectorIndex < 0 {
								AgentConfig.Vector = append(AgentConfig.Vector, vec)
							} else {
								AgentConfig.Vector[vectorIndex] = vec
							}
							break
						}
					}
				}
				currentNeighboursInd = (currentNeighboursInd + 1) %
					len(c.neighbours)
			} else {
				StopReply = true
				Logger.LogSystem("Client ready for reload")
			}
		}
		time.Sleep(2 * time.Second)
	}

}

// Init client
func New(c *AgentConfig.Config) *ServerClient {
	client := &ServerClient{}
	client.init(c)
	return client
}

func (client *ServerClient) init(c *AgentConfig.Config) {
	client.conn = nil
	client.conf = c
	client.neighbours = make([]string, 0)
	client.sectors = make([]int64, 0)
	AgentConfig.Vector = make([]AgentConfig.VectorType, 0)

	if !client.conf.Agent.Master {
		masterServerIndex = rand.Int() % len(client.conf.MasterServers.Hosts)
		currentMasterServerIndex = 0
		Logger.LogSystem(fmt.Sprintf("Use [%s] as master server",
			client.conf.MasterServers.Hosts[masterServerIndex]))
	}

	client.GetSector()
	Logger.LogSystem("Init client side")
}

func (client *ServerClient) Reload(c *AgentConfig.Config) {
	Logger.LogSystem("Client request reload")
	StopRequest = true
	for !StopReply {
	}
	client.init(c)
	StopReply = false
	StopRequest = false
	Logger.LogSystem("Client reloaded")
}

// Function find and return sector index
func (c *ServerClient) GetSector() {
	var (
		sector     []int64 = []int64{}
		sectorsCnt         = len(c.conf.Sectors)
	)
	for i := 0; i < sectorsCnt; i = i + 1 {
		hostsCnt := len(c.conf.Sectors[i].Config.Hosts)
		for j := 0; j < hostsCnt; j = j + 1 {
			hosts := strings.Split(c.conf.Sectors[i].Config.Hosts[j], "|")
			if elExists(hosts, c.conf.Agent.Hostname) {
				sector = append(sector, int64(i))
				Logger.LogInfo(fmt.Sprintf("Sector: %s [%d]",
					c.conf.Sectors[i].Sector, i))
				for l := 1; int64(l) <= c.conf.Agent.HostCnt; l = l + 1 {
					hosts_next := strings.Split(c.conf.Sectors[i].Config.Hosts[(j+l)%
						hostsCnt], "|")
					hosts_prev := strings.Split(
						c.conf.Sectors[i].Config.Hosts[(j-l+hostsCnt)%hostsCnt], "|")
					if !elExists(hosts_next, c.conf.Agent.Hostname) {
						c.AddHost(hosts_next[0], c.conf.Sectors[i].Sector)
						c.AddHost(hosts_prev[0], c.conf.Sectors[i].Sector)
					}
				}
			}
		}
	}
	c.sectors = sector
}

// Function add non-existing host in vector and neighbours array
func (c *ServerClient) AddHost(host string, sector string) {
	var vec AgentConfig.VectorType
	index := getVectorIndexByHost(host)
	if !elExists(c.neighbours, host) {
		c.neighbours = append(c.neighbours, host)
		if index < 0 {
			vec = AgentConfig.VectorType{
				Host:    host,
				Status:  false,
				Sectors: make([]string, 0)}
		} else {
			vec = AgentConfig.Vector[index]
		}
		if !elExists(vec.Sectors, sector) {
			vec.Sectors = append(vec.Sectors,
				sector)
		}

		if index < 0 {
			AgentConfig.Vector = append(AgentConfig.Vector, vec)
		} else {
			AgentConfig.Vector[index] = vec
		}
		Logger.LogInfo(fmt.Sprintf("Added host [%s]", host))
	} else {
		vec = AgentConfig.Vector[index]
		if !elExists(vec.Sectors, sector) {
			vec.Sectors = append(vec.Sectors,
				sector)
		}
		AgentConfig.Vector[index] = vec
		Logger.LogInfo(fmt.Sprintf("Added sector [%s] for host [%s]", sector, host))
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
func (c *ServerClient) sendVector() error {
	var err error = nil
	if c.conf.Agent.Master {
		var vec []AgentConfig.VectorType
		json.Unmarshal(AgentConfig.GetJsonVector(), &vec)
		AgentConfig.MasterVector[c.conf.Agent.Hostname] = vec
		return nil
	}
	if len(c.conf.MasterServers.Hosts) < 1 {
		return fmt.Errorf("Master servers not specifiyed")
	}
	for {
		c.masterconn, err = net.Dial("tcp", fmt.Sprintf("%s:5223",
			c.conf.MasterServers.Hosts[currentMasterServerIndex]))
		if err != nil {
			currentMasterServerIndex = currentMasterServerIndex + 1
			currentMasterServerIndex = currentMasterServerIndex %
				len(c.conf.MasterServers.Hosts)
		} else {
			_, err = c.masterconn.Write([]byte("Meow!\n"))
			_, err = c.masterconn.Write(
				[]byte(fmt.Sprintf("set %s %s\n", c.conf.Agent.Hostname,
					AgentConfig.GetJsonVector())))
			c.masterconn.Close()
			masterServerIndex = currentMasterServerIndex
			break
		}
		if currentMasterServerIndex == masterServerIndex {
			return fmt.Errorf("Could not connect to any of masters")
		}
	}

	return err
}

// Function retur index in vector array if element exist. Else return -1
func getVectorIndexByHost(host string) int {
	for i := 0; i < len(AgentConfig.Vector); i = i + 1 {
		if AgentConfig.Vector[i].Host == host {
			return i
		}
	}
	return -1
}
