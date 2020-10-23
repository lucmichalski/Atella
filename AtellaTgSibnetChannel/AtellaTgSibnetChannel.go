package AtellaTgSibnetChannel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
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

// Function create message for TgSibnet Channel.
func (config *AtellaTgSibnetConfig) newTgSibnetMessage() *tgSibnetMessage {
	local := new(tgSibnetMessage)
	local.Event = "personal"
	local.Usernames = nil
	local.Text = ""
	return local
}

// Function initialize send message (text) via TgSibnet Channel to users,
// specifying in to array in config. It is exportable function
func (config *AtellaTgSibnetConfig) SendMessage(text string, hostname string) (bool,
	error) {
	if config.Disabled {
		return false, nil
	}
	// if !configAtellaTgSibnetChannel.configured {
	// 	return false, fmt.Errorf("SibnetBot is not conigured!")
	// }
	if config.To == nil || len(config.To) < 1 {
		return false, fmt.Errorf("SibnetBot users list are empty")
	}
	msg := config.newTgSibnetMessage()
	msg.Text = fmt.Sprintf("[%s]: %s", hostname, text)
	msg.Usernames = config.To
	result, err := config.sendMessage(*msg)
	// AtellaLogger.LogInfo(fmt.Sprintf("Bot reply %s\n", result))
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
func (config *AtellaTgSibnetConfig) sendMessage(msg tgSibnetMessage) (string, error) {
	conn, err := net.DialTimeout("tcp",
		fmt.Sprintf("%s:%d", config.Address,
		config.Port),
		time.Duration(config.NetTimeout)*time.Second)
	if err != nil {
		return "", fmt.Errorf("%s", err)
	}

	pack := tgSibnetPacket{"sendMessage", msg}
	msg_json, _ := json.Marshal(pack)

	// AtellaLogger.LogInfo(fmt.Sprintf("Send to bot %s\n", string(msg_json)))
	_, err = conn.Write(msg_json)
	if err != nil {
		return "", fmt.Errorf("%s", err)
	}
	reply, _ := bufio.NewReader(conn).ReadString('\n')
	return reply, nil
}
