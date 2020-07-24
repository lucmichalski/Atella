package AgentClient

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"../AgentConfig"
	"../Database"
	"../Logger"
)

var (
	StopRequest bool = false
	StopReply   bool = false
)

type ServerClient struct {
	conn       net.Conn
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

/* Run client */
func (c *ServerClient) Run() {
	var (
		err                   error    = nil
		exit                  bool     = false
		msg                   string   = ""
		msg_map               []string = []string{}
		status                bool     = false
		currenеNeighboursInd  int      = 0
		currentNeighboursAddr string   = c.neighbours[currenеNeighboursInd]
	)
	for {
		if !StopRequest {
			c.conn, err = net.Dial("tcp", fmt.Sprintf("%s:5223",
				currentNeighboursAddr))
			if err != nil {
				Logger.LogError(fmt.Sprintf("%s", err))
				AgentConfig.Vector[currentNeighboursAddr] = false
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
					case "ack":
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
						AgentConfig.Vector[currentNeighboursAddr] = status
						break
					}
				}
			}
			if currenеNeighboursInd == 0 {
				err = insertVector()
				if err != nil {
					Logger.LogError(fmt.Sprintf("%s", err))
				}
			}
			currenеNeighboursInd = (currenеNeighboursInd + 1) %
				len(c.neighbours)
			currentNeighboursAddr = c.neighbours[currenеNeighboursInd]
		} else {
			StopReply = true
			Logger.LogSystem("Client ready for reload")
		}
		time.Sleep(time.Second)
	}

}

/* Init client */
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
	AgentConfig.Vector = make(map[string]bool, 0)
	client.GetSector()
	Logger.LogInfo("Init client side")
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

/* Get sector index */
func (c *ServerClient) GetSector() {
	var (
		sector     []int64 = []int64{}
		sectorsCnt         = len(c.conf.Sectors)
	)
	for i := 0; i < sectorsCnt; i = i + 1 {
		hostsCnt := len(c.conf.Sectors[i].Config.Hosts)
		for j := 0; j < hostsCnt; j = j + 1 {
			hosts := strings.Split(c.conf.Sectors[i].Config.Hosts[j], "|")
			if hostExists(hosts, c.conf.Agent.Hostname) {
				sector = append(sector, int64(i))
				Logger.LogInfo(fmt.Sprintf("Sector: %s [%d]",
					c.conf.Sectors[i].Sector, i))
				for l := 1; int64(l) <= c.conf.Agent.HostCnt; l = l + 1 {
					hosts_next := strings.Split(c.conf.Sectors[i].Config.Hosts[(j+l)%
						hostsCnt], "|")
					hosts_prev := strings.Split(c.conf.Sectors[i].Config.Hosts[(j-l+hostsCnt)%
						hostsCnt], "|")
					if !hostExists(hosts_next, c.conf.Agent.Hostname) {
						c.AddHost(hosts_next[0])
						c.AddHost(hosts_prev[0])
					}
				}
			}
		}
	}
	c.sectors = sector
}

/* Function add non-existing host in hosts array */
func (c *ServerClient) AddHost(host string) {
	if !hostExists(c.neighbours, host) {
		c.neighbours = append(c.neighbours, host)
		AgentConfig.Vector[host] = false
		Logger.LogInfo(fmt.Sprintf("Added host [%s]", host))
	}
}

func hostExists(array []string, item string) bool {
	for i := 0; i < len(array); i = i + 1 {
		if array[i] == item {
			return true
		}
	}
	return false
}

func insertVector() error {
	var err error
	db := Database.GetConnection()
	err = db.Ping()
	if err != nil {
		AgentConfig.PrintJsonVector()
		return err
	}
	// rows, err := db.QueryContext(ctx, "SELECT name FROM users WHERE age = $1", age)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer rows.Close()
	// for rows.Next() {
	// 	var name string
	// 	if err := rows.Scan(&name); err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	fmt.Printf("%s is %d\n", name, age)
	// }
	// if err := rows.Err(); err != nil {
	// 	log.Fatal(err)
	// }
	return nil
}
