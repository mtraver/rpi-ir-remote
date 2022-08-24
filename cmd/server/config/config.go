package config

import (
	"log"
	"os"
	"os/user"
	filepath "path"

	cpb "github.com/mtraver/rpi-ir-remote/cmd/server/configpb"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	currUsr          *user.User
	defautConfigFile string
)

func init() {
	var err error
	currUsr, err = user.Current()
	if err != nil {
		log.Printf("Warning: Failed to get current user so default config file cannot be used. Please provide a path to a config file. Error: %v", err)
	} else {
		defautConfigFile = filepath.Join(currUsr.HomeDir, ".config", "irremote", "irremote.conf.json")
	}
}

func unmarshal(path string) (cpb.Config, error) {
	rawConfig, err := os.ReadFile(path)
	if err != nil {
		return cpb.Config{}, err
	}

	var config cpb.Config
	err = protojson.Unmarshal(rawConfig, &config)
	return config, err
}

func Load(path string) (cpb.Config, error) {
	if path != "" {
		log.Printf("Using config %v", path)
		return unmarshal(path)
	}

	if _, err := os.Stat(defautConfigFile); !os.IsNotExist(err) {
		log.Printf("Using config %v", defautConfigFile)
		return unmarshal(defautConfigFile)
	}

	log.Printf("Using default config")
	return cpb.Config{}, nil
}
