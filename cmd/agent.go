package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"./AgentClient"
	"./AgentConfig"
	"./AgentServer"
	"./Database"
	"./Logger"
)

var (
	conf           *AgentConfig.Config       = nil
	runMode        string                    = "Distributed"
	reportMessage  string                    = "Test"
	reportType     string                    = "Test"
	configFilePath string                    = ""
	configDirPath  string                    = ""
	logLevel       int64                     = 0
	pidFilePath    string                    = "./agent.pid"
	target         string                    = "all"
	client         *AgentClient.ServerClient = nil
)

func handle(c chan os.Signal) {
	for {
		sig := <-c
		Logger.LogSystem(fmt.Sprintf("Receive %s", sig))
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
	flag.Usage = usage
	flag.StringVar(&configFilePath, "config", "",
		"Path to config")
	flag.StringVar(&configDirPath, "configdir", "",
		"Path to config directory")
	flag.StringVar(&runMode, "runmode", "Distributed",
		"Run mode. Possible values:\n\t"+
			"Distributed\n\t"+
			"Send\n\t"+
			"Report\n")
	flag.StringVar(&reportMessage, "msg", "Test",
		"Message. Work only with run mode \"Report\" & report type \"Custom\"")
	flag.StringVar(&reportType, "mode", "Test",
		"Report mode. Possible values:\n\t"+
			"Reboot\n\t"+
			"Custom\n")
	flag.StringVar(&target, "target", "all",
		"Report target. Possible values:\n\t"+
			"All\n\t"+
			"Tgsibnet\n\t"+
			"Mail\n\t"+
			"Graphite\n")
	flag.Parse()
}

func main() {
	var err error = nil
	initFlags()
	conf = AgentConfig.NewConfig()
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

	_, err = os.Stat(conf.Agent.MessagePath)
	if os.IsNotExist(err) {
		os.MkdirAll(conf.Agent.MessagePath, os.ModePerm)
	}

	if strings.ToLower(runMode) == "report" {
		if strings.ToLower(reportType) == "reboot" {
			reportMessage = fmt.Sprintf("Host [%s] has been power-on at [%s]",
				conf.Agent.Hostname, time.Now())
		}
		conf.Report(reportMessage, target)
		os.Exit(0)
	} else if strings.ToLower(runMode) == "send" {
		conf.TryReport()
		os.Exit(0)
	}
	Database.Init(conf)
	Database.Connect()
	conf.SavePid()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	// signal.Notify(c, syscall.SIGINT)
	go handle(c)
	server := AgentServer.New(conf, "0.0.0.0:5223")

	go server.Listen()

	client = AgentClient.New(conf)
	go client.Run()
	for {
		time.Sleep(10 * time.Second)
	}
	//	os.Exit(0)
}
