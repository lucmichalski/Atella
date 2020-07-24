package AgentConfig

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"../Logger"
	"../MailChannel"
	"../TgSibnetChannel"
)

// Abstract Channels configuration.
type ChannelsConfig struct {
	Channel string      `json:"channel"`
	Config  interface{} `json:"config"`
}

type msg struct {
	Target  string `json:"target"`
	Message string `json:"message"`
}

var (
	defaultChannels []string = []string{"tgsibnet", "mail"}
)

// Function return a pseudo-random generated hex-string. 
// String length are specifyed by config (And get to function as n)
func RandomHex(n int64) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Init empty channels configuration
func (conf *Config) Init() {
	var rp interface{}
	Logger.Init(conf.Agent.LogLevel, conf.Agent.LogFile)
	for i := range conf.Channels {
		rp = conf.Channels[i].Config
		switch conf.Channels[i].Channel {
		case "TgSibnet":
			TgSibnetChannel.TgSibnetInit(*rp.(*TgSibnetChannel.TgSibnetConfig))
		case "Mail":
			MailChannel.MailInit(*rp.(*MailChannel.MailConfig), conf.Agent.Hostname)
		default:
			Logger.LogWarning(fmt.Sprintf("Unknown channel %s",
				conf.Channels[i].Channel))
		}
	}
}

// Function call send-report mechanism. Use files created by Report function.
func (conf *Config) TryReport() {
	var (
		err     error  = nil
		message string = ""
		target  string = ""
		res     bool   = true
		m       msg
	)

	files, err := ioutil.ReadDir(conf.Agent.MessagePath)
	if err != nil {
		Logger.LogError(fmt.Sprintf("%s", err))
	}

	for _, file := range files {
		if !file.IsDir() {
			f, err := os.Open(fmt.Sprintf("%s/%s", conf.Agent.MessagePath, file.Name()))
			if err != nil {
				Logger.LogError(fmt.Sprintf("%s", err))
				continue
			}
			data := make([]byte, file.Size())

			for {
				n, err := f.Read(data)
				if err == io.EOF {
					break
				}
				message = message + string(data[:n])
			}
			f.Close()
			err = json.Unmarshal(data, &m)
			if err != nil {
				Logger.LogError(fmt.Sprintf("%s", err))
			}
			Logger.LogInfo(fmt.Sprintf("Read msg - %s [msg: %s|target: %s]", file.Name(), m.Message, m.Target))

			target = strings.ToLower(m.Target)
			if target == "tgsibnet" {
				if conf.Channels["TgSibnet"] != nil {
					res, err = TgSibnetChannel.TgSibnetSendPersonalMessage(m.Message,
						conf.Agent.Hostname)
					if err != nil {
						Logger.LogError(fmt.Sprintf("%s", err))
					}
				}
			} else if target == "mail" {
				if conf.Channels["Mail"] != nil {
					res, err = MailChannel.MailSendMessage(m.Message, conf.Agent.Hostname)
					if err != nil {
						Logger.LogError(fmt.Sprintf("%s", err))
					}
				}
			} else {
				Logger.LogError(fmt.Sprintf("Unsopported channel - %s", target))
				res = true
			}
			if res == true {
				os.Remove(fmt.Sprintf("%s/%s", conf.Agent.MessagePath, file.Name()))
			}
		}
	}
}

// Function save report as a file (filename are random hex string).
func (conf *Config) Report(message string, target string) string {
	var (
		hash string = ""
		path string = ""
		m    *msg   = &msg{
			Message: "",
			Target:  ""}
		file    *os.File = nil
		err     error    = nil
		targets []string = make([]string, 0)
	)
	if strings.ToLower(target) == "all" {
		targets = defaultChannels
	} else {
		targets = append(targets, target)
	}
	for i := 0; i < len(targets); i = i + 1 {
		for {
			hash, _ = RandomHex(conf.Agent.HexLen)
			path = fmt.Sprintf("%s/%s", conf.Agent.MessagePath, hash)
			_, err = os.Stat(path)
			if os.IsNotExist(err) {
				break
			}
		}
		file, err = os.Create(path)
		if err != nil {
			Logger.LogError(fmt.Sprintf("Unable to create file: %s", err))
		}

		defer file.Close()
		m.Message = message
		m.Target = targets[i]
		js, _ := json.Marshal(m)
		file.Write([]byte(js))
		Logger.LogInfo(fmt.Sprintf("File - %s [msg: %s|target: %s]", path, message, targets[i]))
	}
	return hash
}
