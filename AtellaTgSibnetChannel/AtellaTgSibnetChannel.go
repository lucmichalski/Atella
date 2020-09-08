package AtellaTgSibnetChannel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"../AtellaLogger"
)

// Config for Sibnet TG bot. Exportable and used in Agent Config
type AtellaTgSibnetConfig struct {
	Address    string   `json:"address"`
	Port       int16    `json:"port"`
	Protocol   string   `json:"protocol"`
	To         []string `json:"to"`
	Disabled   bool     `json:"disabled"`
	NetTimeout int
}

// Message format for Sibnet TG bot.
type tgSibnetMessage struct {
	Event     string   `json:"event"`
	Usernames []string `json:"usernames"`
	Text      string   `json:"text"`
}

// Packet format for Sibnet TG bot.
type tgSibnetPacket struct {
	Command string          `json:"command"`
	Message tgSibnetMessage `json:"message"`
}

// Configuration for Sibnet TG bot.
type tgSibnetConfig struct {
	host       string
	port       int16
	protocol   string
	to         []string
	configured bool
	disabled   bool
	netTimeout int
}

var (
	// Global config for TG Sibnet Channel.
	configAtellaTgSibnetChannel = newAtellaTgSibnetConfig()
)

// Function initialize TgSibnet Report Channel.
func AtellaTgSibnetInit(c AtellaTgSibnetConfig) {
	setTgSibnetHost(c.Address)
	setTgSibnetPort(c.Port)
	setTgSibnetTo(c.To)
	setTgSibnetDisabled(c.Disabled)
	setTgSibnetTimeout(c.NetTimeout)
	setAtellaTgSibnetConfigured()
	AtellaLogger.LogInfo(fmt.Sprintf("Init TgSibnetBot Channel with params: "+
		"[address: %s | port: %d]",
		c.Address, c.Port))
}

// Function set nee timeout
func setTgSibnetTimeout(timeout int) {
	configAtellaTgSibnetChannel.netTimeout = timeout
}

// Function set channel status.
func setTgSibnetDisabled(disabled bool) {
	configAtellaTgSibnetChannel.disabled = disabled
}

// Function set channel is configured.
func setAtellaTgSibnetConfigured() {
	configAtellaTgSibnetChannel.configured = true
}

// Function set port for connection.
func setTgSibnetPort(port int16) {
	if port != 0 {
		configAtellaTgSibnetChannel.port = port
	}
}

// Function set endpoints for messages.
func setTgSibnetTo(to []string) {
	if to != nil {
		configAtellaTgSibnetChannel.to = to
	}
}

// Function set ip-address(host) for connection.
func setTgSibnetHost(host string) {
	if host != "" {
		configAtellaTgSibnetChannel.host = host
	}
}

// Function create configuration for TgSibnet Channel.
func newAtellaTgSibnetConfig() *tgSibnetConfig {
	local := new(tgSibnetConfig)
	local.host = "localhost"
	local.port = 1
	local.protocol = "tcp"
	local.configured = false
	local.disabled = false
	return local
}

// Function create message for TgSibnet Channel.
func newTgSibnetMessage() *tgSibnetMessage {
	local := new(tgSibnetMessage)
	local.Event = "personal"
	local.Usernames = nil
	local.Text = ""
	return local
}

// Function initialize send message (text) via TgSibnet Channel to users,
// specifying in to array in config. It is exportable function
func AtellaTgSibnetSendPersonalMessage(text string, hostname string) (bool,
	error) {
	if configAtellaTgSibnetChannel.disabled {
		return false, nil
	}
	if !configAtellaTgSibnetChannel.configured {
		return false, fmt.Errorf("SibnetBot is not conigured!")
	}
	if configAtellaTgSibnetChannel.to == nil {
		return false, fmt.Errorf("SibnetBot users list are empty")
	}
	msg := newTgSibnetMessage()
	msg.Text = fmt.Sprintf("[%s]: %s", hostname, text)
	msg.Usernames = configAtellaTgSibnetChannel.to
	result, err := sendMessage(*msg)
	AtellaLogger.LogInfo(fmt.Sprintf("Bot reply %s\n", result))
	if err != nil {
		return false, fmt.Errorf("%s", err)
	}
	ret := false
	if result == "ok" {
		ret = true
	}
	return ret, nil
}

// Function send message (text) via TgSibnet Channel to users, specifying in
// to array in config. It is not exportable function
func sendMessage(msg tgSibnetMessage) (string, error) {
	conn, err := net.DialTimeout("tcp",
		fmt.Sprintf("%s:%d", configAtellaTgSibnetChannel.host,
			configAtellaTgSibnetChannel.port),
		time.Duration(configAtellaTgSibnetChannel.netTimeout)*time.Second)
	if err != nil {
		return "", fmt.Errorf("%s", err)
	}

	pack := tgSibnetPacket{"sendMessage", msg}
	msg_json, _ := json.Marshal(pack)

	AtellaLogger.LogInfo(fmt.Sprintf("Send to bot %s\n", string(msg_json)))
	_, err = conn.Write(msg_json)
	if err != nil {
		return "", fmt.Errorf("%s", err)
	}
	reply, _ := bufio.NewReader(conn).ReadString('\n')
	return reply, nil
}
