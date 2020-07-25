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
		currentNeighboursAddr string   = ""
		vec                   AgentConfig.VectorType
	)

	for {
		if len(c.neighbours) < 1 {
			Logger.LogInfo("No neighbours")
			StopReply = true
		} else {
			currentNeighboursAddr = c.neighbours[currenеNeighboursInd]
			if !StopRequest {
				c.conn, err = net.Dial("tcp", fmt.Sprintf("%s:5223",
					currentNeighboursAddr))
				vec = AgentConfig.Vector[currentNeighboursAddr]
				if err != nil {
					Logger.LogError(fmt.Sprintf("%s", err))
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
							vec.Status = status
							AgentConfig.Vector[currentNeighboursAddr] = vec
							break
						}
					}
				}
				currenеNeighboursInd = (currenеNeighboursInd + 1) %
					len(c.neighbours)
			} else {
				StopReply = true
				Logger.LogSystem("Client ready for reload")
			}
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
	AgentConfig.Vector = make(map[string]AgentConfig.VectorType, 0)
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
						c.AddHost(hosts_next[0], c.conf.Sectors[i].Sector)
						c.AddHost(hosts_prev[0], c.conf.Sectors[i].Sector)
					}
				}
			}
		}
	}
	c.sectors = sector
}

/* Function add non-existing host in hosts array */
func (c *ServerClient) AddHost(host string, sector string) {
	var vec AgentConfig.VectorType
	if !hostExists(c.neighbours, host) {
		c.neighbours = append(c.neighbours, host)
		if _, ok := AgentConfig.Vector[host]; !ok {
			vec = AgentConfig.VectorType{
				Status:  false,
				Sectors: make([]string, 0)}
		} else {
			vec = AgentConfig.Vector[host]
		}
		if !hostExists(vec.Sectors, sector) {
			vec.Sectors = append(vec.Sectors,
				sector)
		}
		AgentConfig.Vector[host] = vec
		Logger.LogInfo(fmt.Sprintf("Added host [%s]", host))
	} else {

		vec = AgentConfig.Vector[host]
		if !hostExists(vec.Sectors, sector) {
			vec.Sectors = append(vec.Sectors,
				sector)
		}
		AgentConfig.Vector[host] = vec
		Logger.LogInfo(fmt.Sprintf("Added sector [%s] for host [%s]", sector, host))
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

func (c *ServerClient) insertVector() error {

	var err error
	err = c.insertVector()
	if err != nil {
		Logger.LogError(fmt.Sprintf("%s", err))
	}

	db := Database.GetConnection()
	err = db.Ping()
	if err != nil {
		AgentConfig.PrintJsonVector()
		for host, mapEl := range AgentConfig.Vector {
			for _, sec := range mapEl.Sectors {
				fmt.Printf("INSERT vector_stat SET host=%s,server=%s,sector=%s,status=%v,timestamp=%d\n\n",
					c.conf.Agent.Hostname, host, sec, mapEl.Status, time.Now().Unix())
			}
		}
		return err
	}
	stmt, err := db.Prepare(
		fmt.Sprintf("INSERT vector_stat SET host=%s,server=%s,status=%v,timestamp=%s",
			"server", "host", false, "date"))
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	return nil
}
