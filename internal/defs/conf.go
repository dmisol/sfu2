package defs

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Conf struct {
	Port     string   `yaml:"port"`
	Hosts    []string `yaml:"hosts,omitempty"`
	Redirect string   `yaml:"redirect,omitempty"`
	BotUrl   string   `yaml:"bot_url,omitempty"`
}

func ReadConf(name string) (c *Conf, err error) {
	b, err := os.ReadFile(name)
	if err != nil {
		return
	}

	c = &Conf{}
	err = yaml.Unmarshal(b, c)
	return
}
