package AgentConfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	_ "net/http"
	_ "net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"../Logger"
	"../MailChannel"
	"../TgSibnetChannel"
	"github.com/influxdata/toml"
	"github.com/influxdata/toml/ast"
)

var (
	sectionDefaults = []string{"agent"}
	envVarRegex = regexp.MustCompile(`\$\{(\w+)\}|\$(\w+)`)

	envVarEscaper = strings.NewReplacer(
		`"`, `\"`,
		`\`, `\\`,
	)
	Pid    int             = 0
	Vector map[string]bool = make(map[string]bool, 0)
)

type AgentConfig struct {
	Hostname     string `json:"hostname"`
	OmitHostname bool   `json:"omit_hostname"`
	LogFile      string `json:"log_file"`
	PidFile      string `json:"pid_file"`
	LogLevel     int64  `json:"log_level"`
	HostCnt      int64  `json:"host_cnt"`
	HexLen       int64  `json:"hex_len"`
	MessagePath  string `json:"message_path"`
}

type DatabaseConfig struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int64  `json:"port"`
	Dbname   string `json:"dbname"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type SectorsConfig struct {
	Sector string        `json:"sector"`
	Config *SectorConfig `json:"config"`
}

type SectorConfig struct {
	Hosts []string `json:"hosts"`
}

type Config struct {
	Agent    *AgentConfig               `json:"AgentSection"`
	Channels map[string]*ChannelsConfig `json:"ChannelsSection"`
	Sectors []*SectorsConfig `json:"SectorsSection"`
	DB      *DatabaseConfig  `json:"DatabaseSection"`
}

func NewConfig() *Config {
	local := &Config{
		Agent: &AgentConfig{
			Hostname:     "",
			OmitHostname: false,
			LogFile:      "/usr/local/mags/logs/mags.log",
			PidFile:      "/usr/local/mags/mags.pid",
			LogLevel:     4,
			HostCnt:      1,
			HexLen:       10,
			MessagePath:  "/usr/local/mags/msg"},
		DB: &DatabaseConfig{
			Type:     "mariadb",
			Host:     "localhost",
			Port:     3306,
			Dbname:   "default",
			User:     "user",
			Password: "password"},
		Channels: make(map[string]*ChannelsConfig),
		Sectors: make([]*SectorsConfig, 0)}
	return local
}

// Function save procces ID to file, specifyied as pidFilePath.
func (c *Config) SavePid() {
	Pid = os.Getpid()
	pidfile, err := os.OpenFile(c.Agent.PidFile,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		Logger.LogFatal(fmt.Sprintf("%s", err))
	}
	defer pidfile.Close()
	pidfile.WriteString(strconv.FormatInt(int64(Pid), 10))
	Logger.LogSystem(fmt.Sprintf("Running with PID %d\n", Pid))
}

func (c *Config) PrintJsonConfig() {
	config_json := c.GetJsonConfig()
	Logger.LogSystem(string(config_json))
}

func (c *Config) GetJsonConfig() []byte {
	config_json, _ := json.Marshal(c)
	return config_json
}

func PrintJsonVector() {
	res := GetJsonVector()
	Logger.LogSystem(string(res))
}

func GetJsonVector() []byte {
	res, _ := json.Marshal(Vector)
	return res
}

func getDefaultConfigPath() (string, error) {
	envfile := os.Getenv("MAGS_CONFIG_PATH")
	homefile := os.ExpandEnv("${HOME}/.mags/mags.conf")
	etcfile := "/etc/mags/mags.conf"

	for _, path := range []string{envfile, homefile, etcfile} {
		if _, err := os.Stat(path); err == nil {
			Logger.LogSystem(fmt.Sprintf("Using config file: %s", path))
			return path, nil
		}
	}

	return "", fmt.Errorf("No config file specified, and could not find one"+
		" in $MAGS_CONFIG_PATH, %s, or %s", homefile, etcfile)
}

func (c *Config) LoadConfig(path string) error {
	var err error
	if path == "" {
		if path, err = getDefaultConfigPath(); err != nil {
			return err
		}
	}
	data, err := loadConfig(path)
	if err != nil {
		return fmt.Errorf("Error loading %s, %s", path, err)
	}

	tbl, err := parseConfig(data)
	if err != nil {
		return fmt.Errorf("Error parsing %s, %s", path, err)
	}

	/* Parse agent table */
	if val, ok := tbl.Fields["agent"]; ok {
		subTable, ok := val.(*ast.Table)
		if !ok {
			return fmt.Errorf("%s: invalid configuration", path)
		}
		if err = toml.UnmarshalTable(subTable, c.Agent); err != nil {
			return fmt.Errorf("Error parsing %s, %s", path, err)
		}
	}

	/* If hostname are not force-override try to get system hostname */
	if !c.Agent.OmitHostname {
		if c.Agent.Hostname == "" {
			hostname, err := os.Hostname()
			if err != nil {
				return err
			}

			c.Agent.Hostname = hostname
		}
	}

	/* Parse database table */
	if val, ok := tbl.Fields["database"]; ok {
		subTable, ok := val.(*ast.Table)
		if !ok {
			return fmt.Errorf("%s: invalid configuration", path)
		}
		if err = toml.UnmarshalTable(subTable, c.DB); err != nil {
			return fmt.Errorf("Error parsing %s, %s", path, err)
		}
	}

	for name, val := range tbl.Fields {
		subTable, ok := val.(*ast.Table)
		if !ok {
			return fmt.Errorf("invalid configuration, error parsing field %q as table", name)
		}

		switch name {
		case "channels":
			for pluginName, pluginVal := range subTable.Fields {
				switch pluginSubTable := pluginVal.(type) {
				case *ast.Table:
					if err = c.addChannel(pluginName, pluginSubTable); err != nil {
						return fmt.Errorf("Error parsing %s, %s", pluginName, err)
					}
				default:
					return fmt.Errorf("Unsupported config format: %s",
						pluginName)
				}
			}
		case "sectors":
			for pluginName, pluginVal := range subTable.Fields {
				switch pluginSubTable := pluginVal.(type) {
				case *ast.Table:
					if err = c.addSector(pluginName, pluginSubTable); err != nil {
						return fmt.Errorf("Error parsing %s, %s", pluginName, err)
					}
				default:
					return fmt.Errorf("Unsupported config format: %s",
						pluginName)
				}
			}
		case "agent", "database":
		default:
			return fmt.Errorf("Error parsing %s, %s", name, err)
		}
	}
	return nil
}

func (c *Config) addChannel(name string, table *ast.Table) error {
	rp := &ChannelsConfig{
		Channel: name,
		Config:  nil}
	switch name {
	case "TgSibnet":
		rp.Config = &TgSibnetChannel.TgSibnetConfig{}

	case "Mail":
		rp.Config = &MailChannel.MailConfig{}

	default:
		rp.Channel = name
		rp.Config = nil
	}
	if rp.Config != nil {
		if err := toml.UnmarshalTable(table, rp.Config); err != nil {
			return fmt.Errorf("Error parsing %s", err)
		}
	}
	c.Channels[name] = rp
	//	c.Channels = append(c.Channels, rp)
	return nil
}

func (c *Config) addSector(name string, table *ast.Table) error {
	rp := &SectorsConfig{
		Sector: name,
		Config: &SectorConfig{
			Hosts: []string{}}}

	if err := toml.UnmarshalTable(table, rp.Config); err != nil {
		return fmt.Errorf("Error parsing %s", err)
	}
	c.Sectors = append(c.Sectors, rp)
	return nil
}

func loadConfig(config string) ([]byte, error) {
	return ioutil.ReadFile(config)
}

func escapeEnv(value string) string {
	return envVarEscaper.Replace(value)
}

func trimBOM(f []byte) []byte {
	return bytes.TrimPrefix(f, []byte("\xef\xbb\xbf"))
}

func parseConfig(contents []byte) (*ast.Table, error) {
	contents = trimBOM(contents)

	parameters := envVarRegex.FindAllSubmatch(contents, -1)
	for _, parameter := range parameters {
		if len(parameter) != 3 {
			continue
		}

		var env_var []byte
		if parameter[1] != nil {
			env_var = parameter[1]
		} else if parameter[2] != nil {
			env_var = parameter[2]
		} else {
			continue
		}

		env_val, result := os.LookupEnv(strings.TrimPrefix(string(env_var), "$"))
		if result {
			env_val = escapeEnv(env_val)
			contents = bytes.Replace(contents, parameter[0], []byte(env_val), 1)
		}
	}

	return toml.Parse(contents)
}
