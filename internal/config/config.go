package config

import (
	"github.com/spf13/viper"
)

// Config represents the global application configuration
type Config struct {
	Port      int         `mapstructure:"port"`
	Log       Log         `mapstructure:"log"`
	ImportDir string      `mapstructure:"import-dir"`
	RabbitMQ  RabbitMQ    `mapstructure:"rabbitmq"`
	MySQL     MySQLConfig `mapstructure:"mysql"`
	LWT       LWT         `mapstructure:"lwt"`
}

// Log represents the logging configuration
type Log struct {
	Level   string `mapstructure:"level"`
	LogBase string `mapstructure:"log-base"`
}

// RabbitMQ represents the RabbitMQ configuration
type RabbitMQ struct {
	URL          string      `mapstructure:"url"`
	Exchange     string      `mapstructure:"exchange"`
	ExchangeType string      `mapstructure:"exchange-type"`
	Queue        RabbitQueue `mapstructure:"queue"`
}

// RabbitQueue represents the RabbitMQ queue configuration
type RabbitQueue struct {
	LwtReq string `mapstructure:"lwt-req"`
	LwtRes string `mapstructure:"lwt-res"`
}

// MySQL represents the MySQL database configuration
type MySQLConfig struct {
	Driver             string `mapstructure:"driver"`
	URL                string `mapstructure:"url"`
	MaxLifeTime        int    `mapstructure:"max-life-time"`
	MaxOpenConnections int    `mapstructure:"max-open-connections"`
	MaxIdleConnections int    `mapstructure:"max-idle-connections"`
}

// LWT represents the LWT template configuration
type LWT struct {
	Template LWTTemplate `mapstructure:"template"`
	Tmp      LWTTemp     `mapstructure:"tmp"`
}

// LWTTemplate represents the LWT template configuration
type LWTTemplate struct {
	Official LWTTemplateCountries `mapstructure:"official"`
	Brief    LWTTemplateBrief     `mapstructure:"brief"`
}

// LWTTemplateCountries represents the LWT template countries configuration
type LWTTemplateCountries struct {
	NL LWTTemplateTypes `mapstructure:"nl"`
	BE LWTTemplateTypes `mapstructure:"be"`
}

// LWTTemplateTypes represents the LWT template types configuration
type LWTTemplateTypes struct {
	Split     string `mapstructure:"split"`
	Amazon    string `mapstructure:"amazon"`
	Ebay      string `mapstructure:"ebay"`
	CDiscount string `mapstructure:"c_discount"`
}

// LWTTemplateBrief represents the LWT template brief configuration
type LWTTemplateBrief struct {
	Split     string `mapstructure:"split"`
	Amazon    string `mapstructure:"amazon"`
	Ebay      string `mapstructure:"ebay"`
	CDiscount string `mapstructure:"c_discount"`
}

// LWTTemp represents the LWT temporary file configuration
type LWTTemp struct {
	Dir string `mapstructure:"dir"`
}

// GlobalConfig is the global configuration instance
var GlobalConfig Config

// InitConfig initializes the global configuration
func InitConfig() (*Config, error) {
	err := viper.Unmarshal(&GlobalConfig)
	if err != nil {
		return nil, err
	}
	return &GlobalConfig, nil
}

// GetConfig returns the global configuration
func GetConfig() *Config {
	return &GlobalConfig
}
