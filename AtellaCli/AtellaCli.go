package AtellaCli

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"../AtellaConfig"
	"../AtellaLogger"
)

var (
	conf              *AtellaConfig.Config = nil
	cmd               string               = ""
	msg               string               = "Test"
	reportType        string               = "Test"
	configFilePath    string               = ""
	configDirPath     string               = ""
	target            string               = "all"
	printVersion      bool                 = false
	printPidFile      bool                 = false
	GitCommit         string               = "unknown"
	GoVersion         string               = "unknown"
	Version           string               = "unknown"
	Service           string               = "Atella-Cli"
	Arch              string               = "unknown"
	Sys               string               = "unknown"
	BinPrefix         string               = "/usr/bin"
	ScriptsPrefix     string               = "/usr/lib/atella/scripts"
	updateVersion     string               = "unknown"
	pkgTemplate       string               = "atella_%s-1_%s.%s"
	masterServerIndex int                  = 0
)

// Function initialize application runtime flags.
func initFlags() {
	Arch = runtime.GOARCH
	flag.Usage = usage
	flag.StringVar(&configFilePath, "config", "",
		"Path to config")
	flag.StringVar(&configDirPath, "config-directory", "",
		"Path to config directory")
	flag.StringVar(&cmd, "cmd", "",
		"Command. Possible values:\n\t"+
			"Send\n\t"+
			"Reload\n\t"+
			"Rotate\n\t"+
			"Update\n\t"+
			"WrapConfig\n\t"+
			"Report")
	flag.StringVar(&msg, "message", "Test",
		"Message. Work only with run mode \"Report\" & report type \"Custom\"")
	flag.StringVar(&reportType, "type", "",
		"Report type. Possible values:\n\t"+
			"Reboot\n\t"+
			"Custom")
	flag.StringVar(&target, "channel", "all",
		"Report channel. Possible values:\n\t"+
			"All\n\t"+
			"Tgsibnet\n\t"+
			"Mail\n\t"+
			"Graphite")
	flag.BoolVar(&printVersion, "version", false, "Print version and exit")
	flag.BoolVar(&printPidFile, "print-pidfile", false,
		"Print pid file path and exit")
	flag.StringVar(&updateVersion, "to-version", "",
		"Version for update")
	flag.Parse()

	AtellaLogger.LogSystem(fmt.Sprintf("Started %s version %s",
		Service, Version))
	if printVersion {
		fmt.Println("Atella")
		fmt.Println("Version:", Version)
		fmt.Println("Arch:", Arch)
		fmt.Println("Packet Sys:", Sys)
		fmt.Println("Git Commit:", GitCommit)
		fmt.Println("Go Version:", GoVersion)
		os.Exit(0)
	} else if printPidFile {
		conf := AtellaConfig.NewConfig()
		err := conf.LoadConfig(configFilePath)
		if err != nil {
			AtellaLogger.LogFatal(fmt.Sprintf("%s", err))
		}
		fmt.Println(conf.Agent.PidFile)
		os.Exit(0)
	}
}

func Command() {
	var err error = nil
	initFlags()
	conf = AtellaConfig.NewConfig()
	err = conf.LoadConfig(configFilePath)
	if err != nil {
		AtellaLogger.LogFatal(fmt.Sprintf("%s", err))
	}
	err = conf.LoadDirectory(configDirPath)
	if err != nil {
		AtellaLogger.LogFatal(fmt.Sprintf("%s", err))
	}

	switch strings.ToLower(cmd) {
	case "report":
		switch strings.ToLower(reportType) {
		case "reboot":
			msg = fmt.Sprintf("Host has been power-on at [%s]", time.Now())
			conf.Report(msg, target)
		case "custom":
			conf.Report(msg, target)
		default:
			AtellaLogger.LogError(fmt.Sprintf("Unknown report type: %s", reportType))
		}
		os.Exit(0)
	case "send":
		pid := conf.GetPid()
		if pid > 0 {
			syscall.Kill(pid, syscall.SIGUSR2)
		}
		os.Exit(0)
	case "reload":
		pid := conf.GetPid()
		if pid > 0 {
			syscall.Kill(pid, syscall.SIGHUP)
		}
		os.Exit(0)
	case "update":
		// TODO: Create environment for master servers from atella config
		if updateVersion != "" {
			for {
				masterAddr := strings.Split(
					conf.MasterServers.Hosts[AtellaConfig.CurrentMasterServerIndex], " ")
				masterconn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:5223",
					masterAddr[0]), time.Duration(conf.Agent.NetTimeout)*time.Second)
				if err != nil {
					AtellaConfig.CurrentMasterServerIndex =
						AtellaConfig.CurrentMasterServerIndex + 1
					AtellaConfig.CurrentMasterServerIndex =
						AtellaConfig.CurrentMasterServerIndex %
							len(conf.MasterServers.Hosts)
				} else {
					masterconn.Close()
					masterServerIndex = AtellaConfig.CurrentMasterServerIndex
					AtellaLogger.LogSystem(fmt.Sprintf("%s using for upgrade",
						masterAddr[0]))
					cmd := exec.Command(fmt.Sprintf("%s/atella-updater.sh",
						ScriptsPrefix),
						masterAddr[0],
						fmt.Sprintf(pkgTemplate, updateVersion, Arch, Sys), Sys)
					err = cmd.Start()
					if err != nil {
						AtellaLogger.LogError("Failed exec cli for update")
						AtellaLogger.LogFatal(fmt.Sprintf("%s", err))
					}
					break
				}
				if AtellaConfig.CurrentMasterServerIndex == masterServerIndex {
					AtellaLogger.LogError("Could not connect to any of masters")
					break
				}
			}

			// err := syscall.Exec(fmt.Sprintf("%s/atella-updater.sh",
			// 	AtellaConfig.BinPrefix),
			// 	[]string{fmt.Sprintf("%s/atella-cli",
			// 		AtellaConfig.BinPrefix), "master.atella.local",
			// 		fmt.Sprintf(pkgTemplate, updateVersion, Arch, Sys), Sys},
			// 	os.Environ())
		} else {
			AtellaLogger.LogError("Version not specifyed")
		}
	case "rotate":
	default:
		AtellaLogger.LogError(fmt.Sprintf("Unknown command: %s", cmd))
	}
}

// Function is a handler for runtime flag -h.
func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [params]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}
