package AtellaLogger

import "log"

type AtellaLogger struct {
	logLevel int64
	logFile  string
}

func New(level int64, file string) *AtellaLogger {
	return &AtellaLogger{}
}

func (logger *AtellaLogger) Init(level int64, file string) {
	logger.setLogLevel(level)
	logger.setLogFile(file)
}

func (logger *AtellaLogger) setLogLevel(level int64) {
	logger.logLevel = level
}

func (logger *AtellaLogger) setLogFile(file string) {
	logger.logFile = file
}

func (logger *AtellaLogger) LogFatal(s string) {
	log.Fatalf("[FATAL]: %s", s)
}

func (logger *AtellaLogger) LogSystem(s string) {
	log.Printf("[SYS]: %s", s)
}

func (logger *AtellaLogger) LogError(s string) {
	if logger.logLevel > 1 {
		log.Printf("[ERROR]: %s", s)
	}
}

func (logger *AtellaLogger) LogWarning(s string) {
	if logger.logLevel > 2 {
		log.Printf("[WARN]: %s", s)
	}
}

func (logger *AtellaLogger) LogInfo(s string) {
	if logger.logLevel > 3 {
		log.Printf("[INFO]: %s", s)
	}
}
