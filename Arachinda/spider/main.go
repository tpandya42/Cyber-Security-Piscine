package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type conf struct {
	url string
	depth	int
	recursion bool
	path 	string
}

func parseArgs(args []string) (conf, error) {
	if len(args) < 1 {
		return conf{}, fmt.Errorf("No URL provided")
	}

	cfg := conf {
		depth: 5,
		recursion: false,
		path: "./data/",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-r":
			cfg.recursion = true

		case "-l":
			if i+1 >= len(args) {
				return conf{}, fmt.Errorf("missing value for -l")
			}
			depth, err := strconv.Atoi(args[i + 1])
			if err != nil {
				return conf{}, fmt.Errorf("invalid depth :%s", args[i+1])
			}
			if depth < 0 {
				return conf{}, fmt.Errorf("invalid depth")	
			}
			cfg.depth = depth
			i++

		case "-p":
			if i+1 >= len(args) {
				return conf{}, fmt.Errorf("missing value for -p")
			}
			cfg.path = args[i + 1]
			i++

		default:
			if strings.HasPrefix(args[i], "-") {
				return conf{}, fmt.Errorf("unknown option: %s", args[i])
			}
			if cfg.url != "" {
				return conf{}, fmt.Errorf("multiple URLs provided")
			}
			cfg.url = args[i]
		}
	}
	if cfg.url == "" {
		return conf{}, fmt.Errorf("no URL provided")
	}
	return cfg, nil
}


func main() {
	// parse flags and url
	cfg, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(cfg)
	// get the html of page
	// get the images if found
}
