package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jalseth/tfplan-origins/internal/terraform"
)

type config struct {
	dir string
	out string
}

func main() {
	var cfg config

	flag.StringVar(&cfg.dir, "dir", ".", "The source directory of the Terraform entrypoint.")
	flag.StringVar(&cfg.out, "out", "tfplan_origins.json", "The path for the output JSON file.")
	flag.Parse()

	if err := realMain(&cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func realMain(cfg *config) error {
	dir, err := filepath.Abs(cfg.dir)
	if err != nil {
		return err
	}

	locs, err := terraform.ParseLocations(os.DirFS("/"), strings.TrimPrefix(dir, "/"))
	if err != nil {
		return err
	}
	for _, loc := range locs {
		loc.File = "/" + loc.File
	}

	by, err := json.MarshalIndent(locs, "", "  ")
	if err != nil {
		return err
	}
	if cfg.out != "" && cfg.out != "-" {
		return os.WriteFile(cfg.out, by, os.ModePerm)
	}
	fmt.Println(string(by))
	return nil
}
