package TgSibnetChannel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	"../Logger"
)

// Config for Sibnet TG bot. Exportable and used in Agent Config
type TgSibnetConfig struct {
	Address  string   `json:"address"`
	Port     int16    `json:"port"`
	Protocol string   `json:"protocol"`
	To       []string `json:"to"`
	Disabled bool     `json:"disbled"`
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
}

var (
	// Global config for TG Sibnet Channel.
	configTgSibnetChannel = newTgSibnetConfig()
)

// Function initialize TgSibnet Report Channel.
func TgSibnetInit(c TgSibnetConfig) {
	setTgSibnetHost(c.Address)
	setTgSibnetPort(c.Port)
	setTgSibnetTo(c.To)
	setTgSibnetDisabled(c.Disabled)
	setTgSibnetConfigured()
	Logger.LogInfo(fmt.Sprintf("Init TgSibnetBot Channel with params: "+
		"[address: %s | port: %d]",
		c.Address, c.Port))
}

// Function set channel status.
func setTgSibnetDisabled(disabled bool) {
	configTgSibnetChannel.disabled = disabled
}

// Function set channel is configured.
func setTgSibnetConfigured() {
	configTgSibnetChannel.configured = true
}

// Function set port for connection.
func setTgSibnetPort(port int16) {
	if port != 0 {
		configTgSibnetChannel.port = port
	}
}

// Function set endpoints for messages.
func setTgSibnetTo(to []string) {
	if to != nil {
		configTgSibnetChannel.to = to
	}
}

// Function set ip-address(host) for connection.
func setTgSibnetHost(host string) {
	if host != "" {
		configTgSibnetChannel.host = host
	}
}

// Function create configuration for TgSibnet Channel.
func newTgSibnetConfig() *tgSibnetConfig {
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
func TgSibnetSendPersonalMessage(text string, hostname string) (bool,
	error) {
	if configTgSibnetChannel.disabled {
		return false, nil
	}
	if !configTgSibnetChannel.configured {
		return false, fmt.Errorf("SibnetBot is not conigured!")
	}
	if configTgSibnetChannel.to == nil {
		return false, fmt.Errorf("SibnetBot users list are empty")
	}
	msg := newTgSibnetMessage()
	msg.Text = fmt.Sprintf("[agent at %s]: %s", hostname, text)
	msg.Usernames = configTgSibnetChannel.to
	result, err := sendMessage(*msg)
	Logger.LogInfo(fmt.Sprintf("Bot reply %s\n", result))
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
	conn, err := net.Dial("tcp", configTgSibnetChannel.host+":"+
		strconv.FormatInt(int64(configTgSibnetChannel.port), 10))
	if err != nil {
		return "", fmt.Errorf("%s", err)
	}

	pack := tgSibnetPacket{"sendMessage", msg}
	msg_json, _ := json.Marshal(pack)

	Logger.LogInfo(fmt.Sprintf("Send to bot %s\n", string(msg_json)))
	_, err = conn.Write(msg_json)
	if err != nil {
		return "", fmt.Errorf("%s", err)
	}
	reply, _ := bufio.NewReader(conn).ReadString('\n')
	return reply, nil
}
