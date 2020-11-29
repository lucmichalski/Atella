package AtellaGraphiteChannel

import (
	"fmt"
	"net"
	"time"
)

// TODO: Send metric via tcp/udp
// (Metric may be specified as a message in other channels)

// Config for Graphite. Exportable and used in Agent Config
type AtellaGraphiteConfig struct {
	Address    string `json:"address"`
	Port       int16  `json:"port"`
	Protocol   string `json:"protocol"`
	Prefix     string `json:"prefix"`
	Disabled   bool   `json:"disabled"`
	NetTimeout int
}

func (config *AtellaGraphiteConfig) Send(text string, hostname string) (bool, error) {
	if config.Disabled {
		return false, nil
	}

	conn, err := net.DialTimeout("tcp",
		fmt.Sprintf("%s:%d", config.Address,
			config.Port),
		time.Duration(config.NetTimeout)*time.Second)
	if err != nil {
		return false, fmt.Errorf("%s", err)
	}
	defer conn.Close()
	metric := fmt.Sprintf("%s.%s.%s", config.Prefix, hostname, text)
  conn.Write([]byte(metric))

	return true, nil
}
