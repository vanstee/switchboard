package switchboard

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Commands map[string]*Command `yaml:"commands"`
	Routes   map[string]*Route   `yaml:"routes"`
}

func ParseConfig(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	c := Config{}
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
