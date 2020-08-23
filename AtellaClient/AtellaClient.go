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
	"../AtellaLogger"
)

var (
	StopRequest       bool = false
	StopReply         bool = false
	masterServerIndex int  = 0
)

type ServerClient struct {
	conn       net.Conn
	masterconn net.Conn
	conf       *AtellaConfig.Config
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
		if len(c.neighbours) < 1 {
			AtellaLogger.LogInfo("No neighbours")
			StopReply = true
		} else {
			currentNeighboursAddrs = strings.Split(c.neighbours[currentNeighboursInd],
				",")
			for _, currentNeighboursAddr = range currentNeighboursAddrs {
				if currentNeighboursInd == 0 {
					err = c.SendToMaster(fmt.Sprintf("set %s %s\n", c.conf.Agent.Hostname,
						AtellaConfig.GetJsonVector()))
					if err != nil {
						AtellaLogger.LogError(fmt.Sprintf("%s", err))
					}
				}
				if !StopRequest {
					c.conn, err = net.DialTimeout("tcp", fmt.Sprintf("%s:5223",
						currentNeighboursAddr),
						time.Duration(c.conf.Agent.NetTimeout)*time.Second)
					vectorIndex := getVectorIndexByHost(currentNeighboursAddr)
					if vectorIndex >= 0 {
						vec = AtellaConfig.Vector[vectorIndex]
					} else {
						vec = AtellaConfig.VectorType{}
						vec.Host = currentNeighboursAddr
						vec.Status = false
					}
					if err != nil {
						AtellaLogger.LogWarning(fmt.Sprintf("%s", err))
						vec.Status = false
					} else {
						exit = false
						connbuf := bufio.NewReader(c.conn)
						err = c.Send(fmt.Sprintf("%s\n", c.conf.Security.Code))
						if err != nil {
							status = false
							exit = true
							AtellaLogger.LogError(fmt.Sprintf("%s", err))
						}
						for {
							message, err := connbuf.ReadString('\n')
							if err != nil {
								if err != io.EOF {
									status = false
									exit = true
									AtellaLogger.LogError(fmt.Sprintf("%s", err))
								}
							}
							if exit {
								c.Close()
								vec.Status = status
								vec.Hostname = hostname
								if vectorIndex < 0 {
									AtellaConfig.Vector = append(AtellaConfig.Vector, vec)
								} else {
									AtellaConfig.Vector[vectorIndex] = vec
								}
								break
							}

							msg = strings.TrimRight(message, "\r\n")
							msgMap = strings.Split(msg, " ")

							AtellaLogger.LogInfo(fmt.Sprintf("Client receive [%s]", msg))
							switch msgMap[0] {
							case "canTalk":
								err = c.Send("hostname\n")
								if err != nil {
									status = false
									exit = true
									AtellaLogger.LogError(fmt.Sprintf("%s", err))
								}
							case "ackhostname":
								if len(msgMap) < 2 {
									exit = true
								} else {
									hostname = msgMap[1]
									err = c.Send(fmt.Sprintf("host %s\n", c.conf.Agent.Hostname))
									if err != nil {
										exit = true
										AtellaLogger.LogError(fmt.Sprintf("%s", err))
									}
								}
							case "ackhost":
								if len(msgMap) < 2 {
									exit = true
								} else {
									if msgMap[1] == c.conf.Agent.Hostname {
										status = true
									} else {
										status = false
									}
									err = c.Send("exit\n")
									if err != nil {
										exit = true
										AtellaLogger.LogError(fmt.Sprintf("%s", err))
									}
								}
							case "Bye!":
								exit = true
							}
						}
					}
				} else {
					StopReply = true
					AtellaLogger.LogSystem("Client ready for reload")
				}
			}
			currentNeighboursInd = (currentNeighboursInd + 1) %
				len(c.neighbours)
		}
		time.Sleep(2 * time.Second)
	}

}

// Init client
func New(c *AtellaConfig.Config) *ServerClient {
	client := &ServerClient{}
	client.init(c)
	return client
}

func (client *ServerClient) init(c *AtellaConfig.Config) {
	client.conn = nil
	client.conf = c
	client.neighbours = make([]string, 0)
	client.sectors = make([]int64, 0)
	AtellaConfig.Vector = make([]AtellaConfig.VectorType, 0)

	if len(client.conf.MasterServers.Hosts) < 1 {
		AtellaConfig.CurrentMasterServerIndex = -1
		AtellaLogger.LogWarning(fmt.Sprintf("Master servers not specifiyed!"))
	} else if !client.conf.Agent.Master {
		masterServerIndex = rand.Int() % len(client.conf.MasterServers.Hosts)
		AtellaConfig.CurrentMasterServerIndex = 0
		AtellaLogger.LogSystem(fmt.Sprintf("Use [%s] as master server",
			client.conf.MasterServers.Hosts[masterServerIndex]))
	}

	client.GetSector()
	AtellaLogger.LogSystem("Init client side")
}

func (client *ServerClient) Reload(c *AtellaConfig.Config) {
	AtellaLogger.LogSystem("Client request reload")
	StopRequest = true
	for !StopReply {
	}
	client.init(c)
	StopReply = false
	StopRequest = false
	AtellaLogger.LogSystem("Client reloaded")
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
			hosts := strings.Split(c.conf.Sectors[i].Config.Hosts[j], " ")
			if elExists(hosts, c.conf.Agent.Hostname) {
				sector = append(sector, int64(i))
				AtellaLogger.LogInfo(fmt.Sprintf("Sector: %s [%d]",
					c.conf.Sectors[i].Sector, i))
				for l := 1; int64(l) <= c.conf.Agent.HostCnt; l = l + 1 {
					hosts_next := strings.Split(c.conf.Sectors[i].Config.Hosts[(j+l)%
						hostsCnt], " ")
					hosts_prev := strings.Split(
						c.conf.Sectors[i].Config.Hosts[(j-l+hostsCnt)%hostsCnt], " ")
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
	var vec AtellaConfig.VectorType
	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		index := getVectorIndexByHost(h)
		if !elExists(c.neighbours, h) {
			c.neighbours = append(c.neighbours, h)
			if index < 0 {
				vec = AtellaConfig.VectorType{
					Host:     h,
					Hostname: "unknown",
					Status:   false,
					Sectors:  make([]string, 0)}
			} else {
				vec = AtellaConfig.Vector[index]
			}
			if !elExists(vec.Sectors, sector) {
				vec.Sectors = append(vec.Sectors,
					sector)
			}
			if index < 0 {
				AtellaConfig.Vector = append(AtellaConfig.Vector, vec)
			} else {
				AtellaConfig.Vector[index] = vec
			}
			AtellaLogger.LogInfo(fmt.Sprintf("Added host [%s]",
				h))
		} else {
			vec = AtellaConfig.Vector[index]
			if !elExists(vec.Sectors, sector) {
				vec.Sectors = append(vec.Sectors,
					sector)
			}
			AtellaConfig.Vector[index] = vec
			AtellaLogger.LogInfo(fmt.Sprintf("Added sector [%s] for host [%s]",
				sector, h))
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
	if c.conf.Agent.Master {
		var vec []AtellaConfig.VectorType
		json.Unmarshal(AtellaConfig.GetJsonVector(), &vec)
		AtellaConfig.MasterVector[c.conf.Agent.Hostname] = vec
		return nil
	}
	if AtellaConfig.CurrentMasterServerIndex < 0 {
		return fmt.Errorf("Master servers not specifiyed")
	}
	for {
		c.masterconn, err = net.Dial("tcp", fmt.Sprintf("%s:5223",
			c.conf.MasterServers.Hosts[AtellaConfig.CurrentMasterServerIndex]))
		if err != nil {
			AtellaConfig.CurrentMasterServerIndex =
				AtellaConfig.CurrentMasterServerIndex + 1
			AtellaConfig.CurrentMasterServerIndex =
				AtellaConfig.CurrentMasterServerIndex %
					len(c.conf.MasterServers.Hosts)
		} else {
			_, err = c.masterconn.Write(
				[]byte(fmt.Sprintf("%s\n", c.conf.Security.Code)))
			_, err = c.masterconn.Write([]byte(query))
			c.masterconn.Close()
			masterServerIndex = AtellaConfig.CurrentMasterServerIndex
			break
		}
		if AtellaConfig.CurrentMasterServerIndex == masterServerIndex {
			return fmt.Errorf("Could not connect to any of masters")
		}
	}

	return err
}

// Function retur index in vector array if element exist. Else return -1
func getVectorIndexByHost(host string) int {
	for i := 0; i < len(AtellaConfig.Vector); i = i + 1 {
		if AtellaConfig.Vector[i].Host == host {
			return i
		}
	}
	return -1
}
