package main

import (
	"github.com/golang/glog"
	"github.com/spf13/viper"
)

type Config struct {
	KafkaBroker []string
	KafkaRootCA string
	KafkaCert   string
	KafkaKey    string
	KafkaTopic  string
	KafkaSSL    bool
}

func ConfigFactory() *Config {
	c := new(Config)

	// Configuration file management
	viper.SetConfigName("config") // name of config file (without extension)
	viper.AddConfigPath("conf/")
	viper.SetConfigType("toml")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		glog.Fatal("Fatal error config file: %s \n")
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
