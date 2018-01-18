package lib

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Options struct {
	Redis struct {
		Uri      string
		Password string
	}
	Mutex struct {
		Prefix     string
		DefaultTTL int
	}
}

func LoadConfig(Filename string) (Options, error) {
	options := Options{}

	// read option file
	config, err := ioutil.ReadFile(Filename)

	if err != nil {
		return options, err
	}

	err = yaml.Unmarshal(config, &options)

	if err != nil {
		return options, err
	}

	return options, nil
}
