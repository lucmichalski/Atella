package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"../../AtellaClient"
	"../../AtellaConfig"
	"../../AtellaDatabase"
	"../../AtellaLogger"
	"../../AtellaServer"
)

var (
	conf           *AtellaConfig.Config       = nil
	configFilePath string                     = ""
	configDirPath  string                     = ""
	client         *AtellaClient.ServerClient = nil
	printVersion   bool                       = false
	GitCommit      string                     = "unknown"
	GoVersion      string                     = "unknown"
	Version        string                     = "unknown"
	Service        string                     = "Atella"
	Arch           string                     = "unknown"
	Sys            string                     = "unknown"
	BinPrefix      string                     = "/usr/bin"
	ScriptsPrefix  string                     = "/usr/lib/atella/scripts"
)

// Interrupts handler
func handle(c chan os.Signal) {
	for {
		sig := <-c
		conf.Logger.LogSystem(fmt.Sprintf("Receive %s [%s]", sig, sig.String()))
		switch sig.String() {
		case "hangup":
			err := conf.LoadConfig(configFilePath)
			if err != nil {
				conf.Logger.LogFatal(fmt.Sprintf("%s", err))
			}
			err = conf.LoadDirectory(configDirPath)
			if err != nil {
				conf.Logger.LogFatal(fmt.Sprintf("%s", err))
			}
			conf.Init()
			conf.PrintJsonConfig()
			client.Reload(conf)
			AtellaDatabase.Reload(conf)
			conf.Logger.LogSystem("Reloaded")
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
	flag.BoolVar(&printVersion, "version", false, "Print version and exit")
	flag.Parse()

	if printVersion {
		fmt.Println("Atella")
		fmt.Println("Version:", Version)
		fmt.Println("Arch:", Arch)
		fmt.Println("Packet Sys:", Sys)
		fmt.Println("Git Commit:", GitCommit)
		fmt.Println("Go Version:", GoVersion)
		os.Exit(0)
	}

	AtellaConfig.GitCommit = GitCommit
	AtellaConfig.GoVersion = GoVersion
	AtellaConfig.Version = Version
	AtellaConfig.Service = Service
	AtellaConfig.Arch = Arch
	AtellaConfig.Sys = Sys
	AtellaConfig.BinPrefix = BinPrefix
	AtellaConfig.ScriptsPrefix = ScriptsPrefix
}

func main() {
	var err error = nil
	initFlags()
	conf = AtellaConfig.NewConfig()
	err = conf.LoadConfig(configFilePath)
	if err != nil {
		logger := AtellaLogger.New(4, "stderr")
		logger.LogFatal(fmt.Sprintf("%s", err))
	}
	err = conf.LoadDirectory(configDirPath)
	if err != nil {
		logger := AtellaLogger.New(4, "stderr")
		logger.LogFatal(fmt.Sprintf("%s", err))
	}
	conf.Init()
	conf.PrintJsonConfig()

	conf.SavePid()
	AtellaDatabase.Init(conf)
	AtellaDatabase.Connect()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	signal.Notify(c, syscall.SIGINT)
	signal.Notify(c, syscall.SIGUSR1)
	signal.Notify(c, syscall.SIGUSR2)

	go handle(c)
	conf.Logger.LogSystem(fmt.Sprintf("Started %s version %s",
		AtellaConfig.Service, AtellaConfig.Version))
	server := AtellaServer.New(conf, "0.0.0.0:5223")
	go server.Listen()
	go server.MasterServer()

	client = AtellaClient.New(conf)
	go client.Run()

	conf.Sender()
}
