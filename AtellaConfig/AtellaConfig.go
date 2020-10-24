package AtellaConfig

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
	"sync"
	"syscall"

	"../AtellaLogger"
	"../AtellaMailChannel"
	"../AtellaTgSibnetChannel"
	"github.com/influxdata/toml"
	"github.com/influxdata/toml/ast"
)

type VectorType struct {
	Host      string   `json:"host"`
	Hostname  string   `json:"hostname"`
	Status    bool     `json:"status"`
	Interval  int64    `json:"interval"`
	Timestamp int64    `json:"timestamp"`
	Sectors   []string `json:"sectors"`
}

var (
	sectionDefaults = []string{"agent"}
	envVarRegex     = regexp.MustCompile(`\$\{(\w+)\}|\$(\w+)`)

	envVarEscaper = strings.NewReplacer(
		`"`, `\"`,
		`\`, `\\`,
	)

	GitCommit     string = "unknown"
	GoVersion     string = "unknown"
	Version       string = "unknown"
	Service       string = "Atella"
	Arch          string = "unknown"
	Sys           string = "unknown"
	BinPrefix     string = "/usr/bin"
	ScriptsPrefix string = "/usr/lib/atella/scripts"
)

type AtellaConfig struct {
	Hostname     string `json:"hostname"`
	OmitHostname bool   `json:"omit_hostname"`
	LogFile      string `json:"log_file"`
	PidFile      string `json:"pid_file"`
	ProcFile     string `json:"proc_file"`
	LogLevel     int64  `json:"log_level"`
	HostCnt      int64  `json:"host_cnt"`
	HexLen       int64  `json:"hex_len"`
	MessagePath  string `json:"message_path"`
	Master       bool   `json:"master"`
	Interval     int64  `json:"interval"`
	NetTimeout   int    `json:"net_timeout"`
}

type SecurityConfig struct {
	Code string `json:"code"`
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

type reporter struct {
	mux      sync.Mutex
	isLocked bool
}

type Config struct {
	Agent                    *AtellaConfig              `json:"AgentSection"`
	Security                 *SecurityConfig            `json:"SecuritySection"`
	Channels                 map[string]*ChannelsConfig `json:"ChannelsSection"`
	Sectors                  []*SectorsConfig           `json:"SectorsSection"`
	DB                       *DatabaseConfig            `json:"DatabaseSection"`
	MasterServers            *MasterServersConfig       `json:"MasterServersSection"`
	reporter                 reporter
	Logger                   *AtellaLogger.AtellaLogger
	Pid                      int
	Vector                   []VectorType
	MasterVector             map[string][]VectorType
	MasterVectorMutex        sync.RWMutex
	CurrentMasterServerIndex int
}

func NewConfig() *Config {
	local := &Config{
		Agent: &AtellaConfig{
			Hostname:     "",
			OmitHostname: false,
			LogFile:      "/var/log/atella/atella.log",
			PidFile:      "/usr/share/atella/atella.pid",
			ProcFile:     "/usr/share/atella/atella.proc",
			LogLevel:     2,
			HostCnt:      1,
			HexLen:       10,
			MessagePath:  "/usr/share/atella/msg",
			Master:       false,
			Interval:     10,
			NetTimeout:   2},
		Security: &SecurityConfig{
			Code: "CodePhrase"},
		DB: &DatabaseConfig{},
		MasterServers: &MasterServersConfig{
			Hosts: make([]string, 0)},
		Channels:                 make(map[string]*ChannelsConfig),
		Sectors:                  make([]*SectorsConfig, 0),
		Logger:                   AtellaLogger.New(4, "stderr"),
		Pid:                      0,
		Vector:                   make([]VectorType, 0),
		MasterVector:             make(map[string][]VectorType, 0),
		MasterVectorMutex:        sync.RWMutex{},
		CurrentMasterServerIndex: 0}

	return local
}

// Function retur vector element in vector array if element exist.
// Else return nil
func (c *Config) GetVectorByHost(host string) (*VectorType, int) {
	for i := 0; i < len(c.Vector); i = i + 1 {
		if c.Vector[i].Host == host {
			return &c.Vector[i], i
		}
	}
	return nil, -1
}

// Function save procces ID to file, specifyied as pidFilePath.
func (c *Config) SavePid() {
	var err error
	c.Pid = os.Getpid()

	_, err = os.Stat(c.Agent.PidFile)
	// Creating path to pid file if it not exist
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
			_, err = os.Stat(fullpath)
			if os.IsNotExist(err) {
				os.MkdirAll(fullpath, 775)
			}
		}
	}

	// Creating path to proc file if it not exist
	_, err = os.Stat(c.Agent.ProcFile)
	if os.IsNotExist(err) {
		path := strings.Split(c.Agent.ProcFile, "/")
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

	// Saving pid and proc
	pidFile, err := os.OpenFile(c.Agent.PidFile,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		c.Logger.LogFatal(fmt.Sprintf("%s", err))
	}

	procFile, err := os.OpenFile(c.Agent.ProcFile,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		c.Logger.LogFatal(fmt.Sprintf("%s", err))
	}
	defer procFile.Close()
	defer pidFile.Close()
	name := strings.Split(os.Args[0], "/")
	pidFile.WriteString(fmt.Sprintf("%d", c.Pid))
	procFile.WriteString(fmt.Sprintf("%s", name[len(name)-1]))
	c.Logger.LogSystem(fmt.Sprintf("Running with PID %d\n", c.Pid))
}

// Function get procces ID from file, specifyied as pidFilePath.
func (c *Config) GetPid() int {
	var pid int = -1
	var cmdLine string = ""
	var name string = ""
	var err error = nil
	file, err := os.Open(c.Agent.PidFile)
	if err != nil {
		c.Logger.LogError(fmt.Sprintf("%s", err))
		return -1
	}
	defer file.Close()
	bytes, err := fmt.Fscanf(file, "%d", &pid)
	if err != nil && err != io.EOF || bytes < 1 {
		c.Logger.LogError(fmt.Sprintf("%s [bytes : %d|file : %s]", err, bytes,
			file.Name()))
		return -1
	}
	procFile, err := os.Open(c.Agent.ProcFile)
	if err != nil {
		c.Logger.LogError(fmt.Sprintf("%s", err))
		return -1
	}
	defer procFile.Close()
	bytes, err = fmt.Fscanf(procFile, "%s", &name)
	if err != nil && err != io.EOF || bytes < 1 {
		c.Logger.LogError(fmt.Sprintf("%s [bytes : %d|file : %s]", err, bytes,
			file.Name()))
		return -1
	}

	cmdFile, err := os.Open(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		c.Logger.LogError(fmt.Sprintf("%s", err))
		return -1
	}
	defer cmdFile.Close()
	bytes, err = fmt.Fscanf(cmdFile, "%s", &cmdLine)
	if err != nil && err != io.EOF || bytes < 1 {
		c.Logger.LogError(fmt.Sprintf("%s [bytes : %d|file : %s]", err, bytes,
			cmdFile.Name()))
		return -1
	}
	c.Logger.LogSystem(fmt.Sprintf("Find PID %d. His command - %s\n",
		pid, cmdLine))
	cmdLineArray := strings.Split(cmdLine, "/")
	cmd := cmdLineArray[len(cmdLineArray)-1]
	cmd = cmd[:len(cmd)-1]
	if cmd != name {
		c.Logger.LogError(fmt.Sprintf(
			"PID not map into agent [%s %s]", cmd, name))
		return -1
	}
	return pid
}

// Function print Config as json format
func (c *Config) PrintJsonConfig() {
	config_json := c.GetJsonConfig()
	c.Logger.LogSystem(string(config_json))
}

// Function return Config as json format
func (c *Config) GetJsonConfig() []byte {
	config_json, _ := json.Marshal(c)
	return config_json
}

// Function print Vector as json format
func (c *Config) PrintJsonVector() {
	res := c.GetJsonVector()
	c.Logger.LogSystem(fmt.Sprintf("Vector %s", string(res)))
}

// Function return Vector as json format
func (c *Config) GetJsonVector() []byte {
	res, _ := json.Marshal(c.Vector)
	return res
}

// Function print MasterVector as json format
func (c *Config) PrintJsonMasterVector() {
	res := c.GetJsonMasterVector()
	c.Logger.LogSystem(fmt.Sprintf("Master Vector %s", string(res)))
}

// Function return MasterVector as json format
func (c *Config) GetJsonMasterVector() []byte {
	c.MasterVectorMutex.RLock()
	res, _ := json.Marshal(c.MasterVector)
	c.MasterVectorMutex.RUnlock()

	return res
}

// Function loads configs from directory
func (c *Config) LoadDirectory(path string) error {
	var err error = nil
	if path == "" {
		if path, err = c.getDefaultConfigDir(); err != nil {
			return err
		}
	}
	walkfn := func(thispath string, info os.FileInfo, _ error) error {
		if info == nil {
			c.Logger.LogWarning(fmt.Sprintf("I don't have permissions to read %s",
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
func (c *Config) getDefaultConfigPath() (string, error) {
	envfile := os.Getenv("ATELLA_CONFIG_PATH")
	homefile := os.ExpandEnv("${HOME}/.atella/atella.conf")
	etcfile := "/etc/atella/atella.conf"

	for _, path := range []string{envfile, homefile, etcfile} {
		if _, err := os.Stat(path); err == nil {
			c.Logger.LogSystem(fmt.Sprintf("Using config file: %s", path))
			return path, nil
		}
	}

	return "", fmt.Errorf("No config file specified, and could not find one"+
		" in $ATELLA_CONFIG_PATH, %s, or %s", homefile, etcfile)
}

// Function return default config dir if it exist
func (c *Config) getDefaultConfigDir() (string, error) {
	envdir := os.Getenv("ATELLA_CONFIG_DIR")
	homedir := os.ExpandEnv("${HOME}/.atella/conf.d")
	etcdir := "/etc/atella/conf.d"

	for _, path := range []string{envdir, homedir, etcdir} {
		if _, err := os.Stat(path); err == nil {
			c.Logger.LogSystem(fmt.Sprintf("Using config directory: %s", path))
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
		if path, err = c.getDefaultConfigPath(); err != nil {
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

	// Parse security table
	if val, ok := tbl.Fields["security"]; ok {
		subTable, ok := val.(*ast.Table)
		if !ok {
			return fmt.Errorf("%s: invalid configuration", path)
		}
		if err = toml.UnmarshalTable(subTable, c.Security); err != nil {
			return fmt.Errorf("Error parsing %s, %s", path, err)
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
			return fmt.Errorf(
				"invalid configuration, error parsing field %q as table", name)
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
		case "agent", "security", "database", "master_servers":
		default:
			return fmt.Errorf("Error parsing %s, %s", name, err)
		}
	}

	_, err = os.Stat(c.Agent.MessagePath)
	if os.IsNotExist(err) {
		syscall.Umask(0)
		os.MkdirAll(c.Agent.MessagePath, 00770|syscall.S_ISGID)
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
		rp.Config = &AtellaTgSibnetChannel.AtellaTgSibnetConfig{
			Address:    "localhost",
			Port:       1,
			Protocol:   "tcp",
			To:         make([]string, 0),
			Disabled:   false,
			NetTimeout: c.Agent.NetTimeout}

	case "Mail":
		rp.Config = &AtellaMailChannel.AtellaMailConfig{
			Address:    "localhost",
			Port:       25,
			Auth:       false,
			Username:   "user",
			Password:   "password",
			From:       "atella@hostname",
			To:         make([]string, 0),
			Disabled:   false,
			NetTimeout: c.Agent.NetTimeout}

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
