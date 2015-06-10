package config

import (
	"encoding/json"
	"errors"
	"flag"
	"github.com/Matir/httpwatch/rules"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

type RepeatedStringFlag []string

// Flag definitons
var configFilename = flag.String("config", "~/.httpwatch", "Configuration file location.")
var logfileName = flag.String("logfile", "", "Logfile for output.")
var interfaces RepeatedStringFlag
var pcapfiles RepeatedStringFlag

// Config represents the whole config
type Config struct {
	Filename   string
	Logfile    string
	Rules      []rules.Rule
	Interfaces []string
	PcapFiles  []string
	Outputs    []outputConfig
	Logger     *log.Logger
}

type outputConfig struct {
	Name    string
	Options map[string]string
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
	c.Filename = name
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

	// Setup logfile from config
	if *logfileName != "" {
		c.Logfile = *logfileName
	} else if c.Logfile == "" {
		c.Logfile = "/dev/stderr"
	}
	if fp, err := os.Create(c.Logfile); err == nil {
		c.Logger = log.New(fp, "httpwatch: ", log.Lshortfile|log.Ltime)
	} else {
		c.Logger = log.New(os.Stderr, "httpwatch: ", log.Lshortfile|log.Ltime)
		c.Logger.Printf("Unable to open %s: %s.  Using stderr instead.\n", c.Logfile, err)
	}
}

func (c *Config) Valid() error {
	if len(c.PcapFiles)+len(c.Interfaces) == 0 {
		return errors.New("Need a pcap or interface!")
	}
	if len(c.Outputs) == 0 {
		return errors.New("Need an output!")
	}
	return nil
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
