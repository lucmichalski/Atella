package AgentConfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	_ "net/http"
	_ "net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"../Logger"
	"../MailChannel"
	"../TgSibnetChannel"
	"github.com/influxdata/toml"
	"github.com/influxdata/toml/ast"
)

type VectorType struct {
	Host    string   `json:"host"`
	Status  bool     `json:"status"`
	Sectors []string `json:"sectors"`
}

var (
	sectionDefaults = []string{"agent"}
	envVarRegex     = regexp.MustCompile(`\$\{(\w+)\}|\$(\w+)`)

	envVarEscaper = strings.NewReplacer(
		`"`, `\"`,
		`\`, `\\`,
	)
	Pid          int                     = 0
	Vector       []VectorType            = make([]VectorType, 0)
	MasterVector map[string][]VectorType = make(map[string][]VectorType, 0)
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
	Master       bool   `json:"master"`
	Interval     int64  `json:"interval"`
	NetTimeout   int    `json:"net_timeout"`
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

type MasterServersConfig struct {
	Hosts []string `json:"hosts"`
}

type SectorConfig struct {
	Hosts []string `json:"hosts"`
}

type Config struct {
	Agent         *AgentConfig               `json:"AgentSection"`
	Channels      map[string]*ChannelsConfig `json:"ChannelsSection"`
	Sectors       []*SectorsConfig           `json:"SectorsSection"`
	DB            *DatabaseConfig            `json:"DatabaseSection"`
	MasterServers *MasterServersConfig       `json:"MasterServersSection"`
}

func NewConfig() *Config {
	local := &Config{
		Agent: &AgentConfig{
			Hostname:     "",
			OmitHostname: false,
			LogFile:      "/usr/share/atella/logs/atella.log",
			PidFile:      "/usr/share/atella/atella.pid",
			LogLevel:     2,
			HostCnt:      1,
			HexLen:       10,
			MessagePath:  "/usr/share/atella/msg",
			Master:       false,
			Interval:     10,
			NetTimeout:   2},
		DB: &DatabaseConfig{},
		MasterServers: &MasterServersConfig{
			Hosts: make([]string, 0)},
		Channels: make(map[string]*ChannelsConfig),
		Sectors:  make([]*SectorsConfig, 0)}

	return local
}

// Function save procces ID to file, specifyied as pidFilePath.
func (c *Config) SavePid() {
	Pid = os.Getpid()

	_, err := os.Stat(c.Agent.PidFile)
	if os.IsNotExist(err) {
		path := strings.Split(c.Agent.PidFile, "/")
		fullpath := "/"
		for i := 0; i < len(path)-1; i = i + 1 {
			if path[i] != "" && path[i] != "/" {
				if fullpath == "" {
					fullpath = fmt.Sprintf("/%s", path[i])
				} else {
					fullpath = fmt.Sprintf("%s%s/", fullpath, path[i])
				}
			}
			_, err := os.Stat(fullpath)
			if os.IsNotExist(err) {
				os.MkdirAll(fullpath, 775)
			}
		}
	}

	file, err := os.OpenFile(c.Agent.PidFile,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		Logger.LogFatal(fmt.Sprintf("%s", err))
	}
	defer file.Close()
	name := strings.Split(os.Args[0], "/")
	file.WriteString(fmt.Sprintf("%d %s", Pid, name[len(name)-1]))
	Logger.LogSystem(fmt.Sprintf("Running with PID %d\n", Pid))
}

// Function get procces ID from file, specifyied as pidFilePath.
func (c *Config) GetPid() int {
	var pid int = -1
	var cmdLine string = ""
	var name string = ""
	file, err := os.Open(c.Agent.PidFile)
	if err != nil {
		Logger.LogError(fmt.Sprintf("%s", err))
		return -1
	}
	defer file.Close()
	bytes, err := fmt.Fscanf(file, "%d %s", &pid, &name)
	if err != nil && err != io.EOF || bytes < 1 {
		Logger.LogError(fmt.Sprintf("%s [bytes : %d|file : %s]", err, bytes,
			file.Name()))
		return -1
	}
	cmdFile, err := os.Open(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		Logger.LogError(fmt.Sprintf("%s", err))
		return -1
	}
	defer cmdFile.Close()
	bytes, err = fmt.Fscanf(cmdFile, "%s", &cmdLine)
	if err != nil && err != io.EOF || bytes < 1 {
		Logger.LogError(fmt.Sprintf("%s [bytes : %d|file : %s]", err, bytes,
			cmdFile.Name()))
		return -1
	}
	Logger.LogSystem(fmt.Sprintf("Find PID %d. His command - %s\n", pid, cmdLine))
	cmdLineArray := strings.Split(cmdLine, "/")
	cmd := cmdLineArray[len(cmdLineArray)-1]
	cmd = cmd[:len(cmd)-1]
	if cmd != name {
		Logger.LogError(fmt.Sprintf(
			"PID not map into agent [%s %s]", cmd, name))
		return -1
	}
	return pid
}

// Function print Config as json format
func (c *Config) PrintJsonConfig() {
	config_json := c.GetJsonConfig()
	Logger.LogSystem(string(config_json))
}

// Function return Config as json format
func (c *Config) GetJsonConfig() []byte {
	config_json, _ := json.Marshal(c)
	return config_json
}

// Function print Vector as json format
func PrintJsonVector() {
	res := GetJsonVector()
	Logger.LogSystem(fmt.Sprintf("Vector %s", string(res)))
}

// Function return Vector as json format
func GetJsonVector() []byte {
	res, _ := json.Marshal(Vector)
	return res
}

// Function print MasterVector as json format
func PrintJsonMasterVector() {
	res := GetJsonMasterVector()
	Logger.LogSystem(fmt.Sprintf("Master Vector %s", string(res)))
}

// Function return MasterVector as json format
func GetJsonMasterVector() []byte {
	res, _ := json.Marshal(MasterVector)
	return res
}

// Function loads configs from directory
func (c *Config) LoadDirectory(path string) error {
	var err error = nil
	if path == "" {
		if path, err = getDefaultConfigDir(); err != nil {
			return err
		}
	}
	walkfn := func(thispath string, info os.FileInfo, _ error) error {
		if info == nil {
			Logger.LogWarning(fmt.Sprintf("I don't have permissions to read %s",
				thispath))
			return nil
		}

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), "..") {
				return filepath.SkipDir
			}
			return nil
		}

		name := info.Name()
		if len(name) < 6 || name[len(name)-5:] != ".conf" {
			return nil
		}
		err := c.LoadConfig(thispath)
		if err != nil {
			return err
		}
		return nil
	}
	return filepath.Walk(path, walkfn)
}

// Function return default config path if it exist
func getDefaultConfigPath() (string, error) {
	envfile := os.Getenv("ATELLA_CONFIG_PATH")
	homefile := os.ExpandEnv("${HOME}/.atella/atella.conf")
	etcfile := "/etc/atella/atella.conf"

	for _, path := range []string{envfile, homefile, etcfile} {
		if _, err := os.Stat(path); err == nil {
			Logger.LogSystem(fmt.Sprintf("Using config file: %s", path))
			return path, nil
		}
	}

	return "", fmt.Errorf("No config file specified, and could not find one"+
		" in $ATELLA_CONFIG_PATH, %s, or %s", homefile, etcfile)
}

// Function return default config dir if it exist
func getDefaultConfigDir() (string, error) {
	envdir := os.Getenv("ATELLA_CONFIG_DIR")
	homedir := os.ExpandEnv("${HOME}/.atella/conf.d")
	etcdir := "/etc/atella/conf.d"

	for _, path := range []string{envdir, homedir, etcdir} {
		if _, err := os.Stat(path); err == nil {
			Logger.LogSystem(fmt.Sprintf("Using config directory: %s", path))
			return path, nil
		}
	}

	return "", fmt.Errorf("No config dir specified, and could not find one"+
		" in $ATELLA_CONFIG_DIR, %s, or %s", homedir, etcdir)
}

// Function loads configs from path
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

	// Parse agent table
	if val, ok := tbl.Fields["agent"]; ok {
		subTable, ok := val.(*ast.Table)
		if !ok {
			return fmt.Errorf("%s: invalid configuration", path)
		}
		if err = toml.UnmarshalTable(subTable, c.Agent); err != nil {
			return fmt.Errorf("Error parsing %s, %s", path, err)
		}
	}

	// If hostname are not force-override try to get system hostname
	if !c.Agent.OmitHostname {
		if c.Agent.Hostname == "" {
			hostname, err := os.Hostname()
			if err != nil {
				return err
			}

			c.Agent.Hostname = hostname
		}
	}

	// Parse database table
	if val, ok := tbl.Fields["database"]; ok {
		subTable, ok := val.(*ast.Table)
		if !ok {
			return fmt.Errorf("%s: invalid configuration", path)
		}
		if err = toml.UnmarshalTable(subTable, c.DB); err != nil {
			return fmt.Errorf("Error parsing %s, %s", path, err)
		}
	}

	// Parse master_servers table
	if val, ok := tbl.Fields["master_servers"]; ok {
		subTable, ok := val.(*ast.Table)
		if !ok {
			return fmt.Errorf("%s: invalid configuration", path)
		}
		if err = toml.UnmarshalTable(subTable, c.MasterServers); err != nil {
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
		case "agent", "database", "master_servers":
		default:
			return fmt.Errorf("Error parsing %s, %s", name, err)
		}
	}
	return nil
}

// Function add channel config into channels section
func (c *Config) addChannel(name string, table *ast.Table) error {
	rp := &ChannelsConfig{
		Channel: name,
		Config:  nil}
	switch name {
	case "TgSibnet":
		rp.Config = &TgSibnetChannel.TgSibnetConfig{NetTimeout: c.Agent.NetTimeout}

	case "Mail":
		rp.Config = &MailChannel.MailConfig{NetTimeout: c.Agent.NetTimeout}

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

// Function add sector config into sectors section
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
