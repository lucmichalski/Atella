package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"../../AtellaClient"
	"../../AtellaConfig"
	"../../AtellaServer"
	"../../Database"
	"../../Logger"
)

var (
	conf             *AtellaConfig.Config       = nil
	runMode          string                     = "Distributed"
	reportMessage    string                     = "Test"
	reportType       string                     = "Test"
	configFilePath   string                     = ""
	configDirPath    string                     = ""
	target           string                     = "all"
	client           *AtellaClient.ServerClient = nil
	printVersion     bool                       = false
	printPidFilePath bool                       = false
	GitCommit        string                     = "unknown"
	GoVersion        string                     = "unknown"
	Version          string                     = "unknown"
	Service          string                     = "Atella"
	Arch             string                     = "unknown"
	Sys              string                     = "unknown"
)

// Interrupts handler
func handle(c chan os.Signal) {
	for {
		sig := <-c
		Logger.LogSystem(fmt.Sprintf("Receive %s [%s]", sig, sig.String()))
		switch sig.String() {
		case "hangup":
			err := conf.LoadConfig(configFilePath)
			if err != nil {
				Logger.LogFatal(fmt.Sprintf("%s", err))
			}
			conf.Init()
			conf.PrintJsonConfig()
			client.Reload(conf)
			Database.Reload(conf)
			Logger.LogSystem("Reloaded")
		case "interrupt":
			os.Exit(0)
		case "user defined signal 1":
			conf.Send()
		case "user defined signal 2":
			conf.Send()
		default:
		}
	}
}

// Function is a handler for runtime flag -h.
func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [params]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

// Function initialize application runtime flags.
func initFlags() {
	Arch = runtime.GOARCH
	flag.Usage = usage
	flag.StringVar(&configFilePath, "config", "",
		"Path to config")
	flag.StringVar(&configDirPath, "config-directory", "",
		"Path to config directory")
	flag.StringVar(&runMode, "run-mode", "Distributed",
		"Run mode. Possible values:\n\t"+
			"Distributed\n\t"+
			"Send\n\t"+
			"Reload\n\t"+
			"Report\n")
	flag.StringVar(&reportMessage, "report-message", "Test",
		"Message. Work only with run mode \"Report\" & report type \"Custom\"")
	flag.StringVar(&reportType, "report-type", "Test",
		"Report type. Possible values:\n\t"+
			"Reboot\n\t"+
			"Custom\n")
	flag.StringVar(&target, "report-target", "all",
		"Report target. Possible values:\n\t"+
			"All\n\t"+
			"Tgsibnet\n\t"+
			"Mail\n\t"+
			"Graphite\n")
	flag.BoolVar(&printVersion, "version", false, "Print version and exit")
	flag.BoolVar(&printPidFilePath, "print-pid-file-path", false,
		"Print pid file path and exit")
	flag.Parse()

	if printVersion {
		fmt.Println("Atella")
		fmt.Println("Version:", Version)
		fmt.Println("Arch:", Arch)
		fmt.Println("Packet Sys:", Sys)
		fmt.Println("Git Commit:", GitCommit)
		fmt.Println("Go Version:", GoVersion)
		os.Exit(0)
	} else if printPidFilePath {
		conf := AtellaConfig.NewConfig()
		err := conf.LoadConfig(configFilePath)
		if err != nil {
			Logger.LogFatal(fmt.Sprintf("%s", err))
		}
		fmt.Println(conf.Agent.PidFile)
		os.Exit(0)
	}

	AtellaConfig.GitCommit = GitCommit
	AtellaConfig.GoVersion = GoVersion
	AtellaConfig.Version = Version
	AtellaConfig.Service = Service
	AtellaConfig.Arch = Arch
	AtellaConfig.Sys = Sys
}

func main() {
	var err error = nil
	initFlags()
	conf = AtellaConfig.NewConfig()
	err = conf.LoadConfig(configFilePath)
	if err != nil {
		Logger.LogFatal(fmt.Sprintf("%s", err))
	}
	err = conf.LoadDirectory(configDirPath)
	if err != nil {
		Logger.LogFatal(fmt.Sprintf("%s", err))
	}
	conf.Init()
	conf.PrintJsonConfig()

	if strings.ToLower(runMode) == "report" {
		if strings.ToLower(reportType) == "reboot" {
			reportMessage = fmt.Sprintf("Host has been power-on at [%s]", time.Now())
		}
		conf.Report(reportMessage, target)
		os.Exit(0)
	} else if strings.ToLower(runMode) == "send" {
		pid := conf.GetPid()
		if pid > 0 {
			syscall.Kill(pid, syscall.SIGUSR2)
		}
		os.Exit(0)
	} else if strings.ToLower(runMode) == "reload" {
		pid := conf.GetPid()
		if pid > 0 {
			syscall.Kill(pid, syscall.SIGHUP)
		}
		os.Exit(0)
	}

	conf.SavePid()
	Database.Init(conf)
	Database.Connect()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	signal.Notify(c, syscall.SIGINT)
	signal.Notify(c, syscall.SIGUSR1)
	signal.Notify(c, syscall.SIGUSR2)

	go handle(c)
	Logger.LogSystem(fmt.Sprintf("Started Atella v%s", AtellaConfig.Version))
	server := AtellaServer.New(conf, "0.0.0.0:5223")
	go server.Listen()
	go server.MasterServer()

	client = AtellaClient.New(conf)
	go client.Run()

	go conf.Sender()

	for {
		time.Sleep(time.Minute)
	}
	//	os.Exit(0)
}
