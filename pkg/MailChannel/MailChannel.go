package MailChannel

import (
	"fmt"
	"net/smtp"
	"regexp"

	"../Logger"
)

// Mail message format.
type mailMessage struct {
	Emails []string `json:"emails"`
	Text   string   `json:"text"`
}

// Mail Channel configuration.
type MailConfig struct {
	Address  string   `json:"address"`
	Port     int16    `json:"port"`
	Auth     bool     `json:"auth"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	From     string   `json:"from"`
	To       []string `json:"to"`
	Disabled bool     `json:"disbled"`
}

// Mail Channel configuration.
type mailConfig struct {
	address    string
	port       int16
	auth       bool
	username   string
	password   string
	from       string
	to         []string
	configured bool
	disabled   bool
}

var (
	// Global config for Mail Channel.
	configMailChannel = newMailConfig()
)

// Function initialize Mail Report Channel.
func MailInit(c MailConfig, hostname string) {
	setMailHost(c.Address)
	setMailPort(c.Port)
	setMailAuth(c.Auth)
	setMailUsername(c.Username)
	setMailPassword(c.Password)
	setMailFrom(c.From)
	setMailTo(c.To)
	setMailDisabled(c.Disabled)
	setMailConfigured()

	re := regexp.MustCompile(`@hostname$`)
	configMailChannel.from = re.ReplaceAllString(configMailChannel.from,
		fmt.Sprintf("@%s", hostname))
	Logger.LogInfo(fmt.Sprintf("Init Mail Channel with params: [address: %s | port: %d]",
		c.Address, c.Port))
}

// Function set channel status.
func setMailDisabled(disabled bool) {
	configMailChannel.disabled = disabled
}

// Function set port for connection.
func setMailPort(port int16) {
	if port != 0 {
		configMailChannel.port = port
	}
}

// Function set ip-address(host) for connection.
func setMailHost(host string) {
	if host != "" {
		configMailChannel.address = host
	}
}

// Function set from field for connection.
func setMailFrom(from string) {
	if from != "" {
		configMailChannel.from = from
	}
}

// Function set to field for connection.
func setMailTo(to []string) {
	if to != nil {
		configMailChannel.to = to
	}
}

// Function set username for connection.
func setMailUsername(username string) {
	if username != "" {
		configMailChannel.username = username
	}
}

// Function set auth for connection.
func setMailAuth(auth bool) {
	configMailChannel.auth = auth
}

// Function set password for connection.
func setMailPassword(password string) {
	configMailChannel.password = password
}

// Function set channel is configured.
func setMailConfigured() {
	configMailChannel.configured = true
}

func newMailConfig() *mailConfig {
	local := new(mailConfig)
	local.address = "localhost"
	local.port = 25
	local.auth = false
	local.username = "user"
	local.password = "password"
	local.from = "mags@hostname"
	local.configured = false
	local.disabled = false
	return local
}

// Function create message for Mail Channel.
func newMailMessage() *mailMessage {
	local := new(mailMessage)
	local.Emails = configMailChannel.to
	local.Text = ""
	return local
}

func MailSendMessage(text string, hostname string) (bool, error) {
	if configMailChannel.disabled {
		return false, nil
	}
	if !configMailChannel.configured {
		return false, fmt.Errorf("Mail is not conigured!")
	}
	if configMailChannel.to == nil {
		return false, fmt.Errorf("Mail users list are empty")
	}
	msg := newMailMessage()
	msg.Text = fmt.Sprintf("Subject: Message from agent at %s\n\r%s",
		hostname, text)
	msg.Emails = configMailChannel.to
	err := sendMessage(*msg)
	if err != nil {
		return false, fmt.Errorf("%s", err)
	}
	return true, nil
}

// Function send message (text) via Mail Channel to user's emails , specifying
// in Emails array. It is not exportable function
func sendMessage(msg mailMessage) error {
	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", configMailChannel.address, configMailChannel.port),
		nil,
		configMailChannel.from,
		msg.Emails,
		[]byte(msg.Text))
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	return nil
}
