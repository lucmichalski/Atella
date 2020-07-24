package Database

import (
	"database/sql"
	"fmt"

	"../AgentConfig"
	"../Logger"
	_ "github.com/go-sql-driver/mysql"
)

var (
	base *sql.DB             = nil
	conf *AgentConfig.Config = nil
)

func Init(c *AgentConfig.Config) {
  conf = c
	Logger.LogInfo(fmt.Sprintf("Init db with [%s:%s@%s:%d/%s",
	conf.DB.User, conf.DB.Password,conf.DB.Host,conf.DB.Port, conf.DB.Dbname))

}

func Reload(c *AgentConfig.Config) {
	base.Close()
  conf = c
	Logger.LogInfo(fmt.Sprintf("Reload db with [%s:%s@%s:%d/%s",
	conf.DB.User, conf.DB.Password,conf.DB.Host,conf.DB.Port, conf.DB.Dbname))
}

func Connect() {
	var err error = nil
	if conf.DB.Type == "mysql" {
		base, err = sql.Open("mysql", "user:password@tcp(127.0.0.1:3306)/hello")
	}
	if err != nil {
		Logger.LogError(fmt.Sprintf("%s", err))
	}
}

func GetConnection() *sql.DB {
	return base
}