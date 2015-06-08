package config

import (
	"encoding/json"
	"flag"
	"github.com/Matir/httpwatch/rules"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

type RepeatedStringFlag []string

// Flag definitons
var configFilename = flag.String("config", "~/.httpwatch", "Configuration file location.")
var interfaces RepeatedStringFlag
var pcapfiles RepeatedStringFlag

// Config represents the whole config
type Config struct {
	Filename   string
	Rules      []rules.Rule
	Interfaces []string
	PcapFiles  []string
}

func (c *Config) ParseConfigFile(name string) {
	fp, err := os.Open(name)
	if err != nil {
		panic(err)
	}

	buf, err := ioutil.ReadAll(fp)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(buf, c)
	if err != nil {
		panic(err)
	}
}

func (c *Config) Init() {
	if !flag.Parsed() {
		flag.Parse()
	}
	c.ParseConfigFile(replaceUserdir(*configFilename))
	if len(interfaces) > 0 {
		c.Interfaces = interfaces
	}
	if len(pcapfiles) > 0 {
		c.PcapFiles = pcapfiles
	}
}

func (rs *RepeatedStringFlag) String() string {
	return strings.Join(*rs, ", ")
}

func (rs *RepeatedStringFlag) Set(value string) error {
	*rs = append(*rs, value)
	return nil
}

func replaceUserdir(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	u, err := user.Current()
	if err != nil {
		return path
	}
	return filepath.Join(u.HomeDir, path[2:])
}

func init() {
	flag.Var(&interfaces, "interfaces", "Interfaces to listen on.")
	flag.Var(&pcapfiles, "pcap", "PCAP Files to parse.")
}
