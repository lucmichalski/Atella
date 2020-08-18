package AtellaCli

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	"../AtellaConfig"
	"../AtellaLogger"
)

var (
	conf           *AtellaConfig.Config = nil
	cmd            string               = ""
	msg            string               = "Test"
	reportType     string               = "Test"
	configFilePath string               = ""
	configDirPath  string               = ""
	target         string               = "all"
	printVersion   bool                 = false
	printPidFile   bool                 = false
	GitCommit      string               = "unknown"
	GoVersion      string               = "unknown"
	Version        string               = "unknown"
	Service        string               = "Atella-Cli"
	Arch           string               = "unknown"
	Sys            string               = "unknown"
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
