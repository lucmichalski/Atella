package AtellaLogger

import "log"

var (
	logLevel int64  = 4
	logFile  string = "stdout"
)

func Init(level int64, file string) {
	setLogLevel(level)
	setLogFile(file)
}

func setLogLevel(level int64) {
	logLevel = level
}

func setLogFile(file string) {
	logFile = file
}

func LogFatal(s string) {
	log.Fatalf("[FATAL]: %s", s)
}

func LogSystem(s string) {
	log.Printf("[SYS]: %s", s)
}

func LogError(s string) {
	if logLevel > 1 {
		log.Printf("[ERROR]: %s", s)
	}
}

func LogWarning(s string) {
	if logLevel > 2 {
		log.Printf("[WARN]: %s", s)
	}
}

func LogInfo(s string) {
	if logLevel > 3 {
		log.Printf("[INFO]: %s", s)
	}
}
