package main

import (
	"github.com/golang/glog"
	"github.com/spf13/viper"
)

// Config go-policy configuration structure
type Config struct {
	KafkaBroker []string
	KafkaRootCA string
	KafkaCert   string
	KafkaKey    string
	KafkaTopic  string
	KafkaSSL    bool
}

// NewConfig Creates a new configuration struct, return a *Config
func NewConfig() *Config {
	c := new(Config)

	// Configuration file management
	// name of config file (without extension)
	viper.SetConfigName("config")
	viper.AddConfigPath("conf/")
	viper.SetConfigType("toml")
	// Find and read the config file
	err := viper.ReadInConfig()
	if err != nil {
		// Handle errors reading the config file
		glog.Fatalf("Fatal error config file: %s", err.Error())
	}

	c.loadConfig()

	return c
}

func (c *Config) loadConfig() {
	k := viper.Sub("kafka")
	c.KafkaBroker = k.GetStringSlice("brokers")
	c.KafkaRootCA = k.GetString("rootca")
	c.KafkaCert = k.GetString("cert")
	c.KafkaKey = k.GetString("key")
	c.KafkaTopic = k.GetString("topic")
	c.KafkaSSL = k.GetBool("ssl")

}
