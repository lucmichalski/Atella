package Database

import (
	"database/sql"
	"fmt"

	"../AtellaConfig"
	"../Logger"
	_ "github.com/go-sql-driver/mysql"
)

var (
	base *sql.DB             = nil
	conf *AtellaConfig.Config = nil
)

func Init(c *AtellaConfig.Config) {
	conf = c
	if conf.DB.Type != "" {
		Logger.LogInfo(fmt.Sprintf("Init db with [%s:%s@%s:%d/%s]",
			conf.DB.User, conf.DB.Password, conf.DB.Host, conf.DB.Port, conf.DB.Dbname))
	} else {
		Logger.LogWarning("Database section not defined")
	}
}

func Reload(c *AtellaConfig.Config) {
	if base != nil {
		base.Close()
	}
	conf = c
	if conf.DB.Type != "" {
		Logger.LogInfo(fmt.Sprintf("Reload db with [%s:%s@%s:%d/%s",
			conf.DB.User, conf.DB.Password, conf.DB.Host, conf.DB.Port, conf.DB.Dbname))
	} else {
		Logger.LogWarning("Database section not defined")
	}
}

func Connect() {
	var err error = nil
	if conf.DB.Type == "" {
		return
	}
	if conf.DB.Type == "mysql" {
		base, err = sql.Open(conf.DB.Type, fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			conf.DB.User, conf.DB.Password, conf.DB.Host, conf.DB.Port, conf.DB.Dbname))
	}
	if err != nil {
		Logger.LogError(fmt.Sprintf("%s", err))
		base = nil
	}
}

func GetConnection() *sql.DB {
	return base
}

func SelectQuery(q string) (int, error) {
	var (
		err   error = nil
		count int   = -1
	)
	if base == nil {
		return count, fmt.Errorf("Database does not exist")
	}
	err = base.Ping()
	if err != nil {
		return count, err
	}
	return count, nil
}

func InsertQuery(q string) error {
	return nil
}

func UpdateQuery(q string) error {
	return nil
}

func Migrate() {

}
