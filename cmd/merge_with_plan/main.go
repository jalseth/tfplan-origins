package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/jalseth/tfplan-origins/internal/terraform"
)

type config struct {
	locs string
	plan string
	out  string
}

func main() {
	var cfg config

	flag.StringVar(&cfg.locs, "locs", "tfplan_origins.json", "The path to the locations file. Accepts an empty string for stdin.")
	flag.StringVar(&cfg.plan, "plan", "plan.json", "The path to the input Terraform plan. Accepts an empty string for stdin.")
	flag.StringVar(&cfg.out, "out", "", "The path to write the modified Terraform plan to. Accepts an empty string for stdout.")
	flag.Parse()

	if err := realMain(&cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func realMain(cfg *config) error {
	if stdinout(cfg.locs) && stdinout(cfg.plan) {
		return fmt.Errorf("locations and plan cannot both be read from stdin")
	}

	locs, err := readJSON[terraform.Locations](cfg.locs)
	if err != nil {
		return fmt.Errorf("read locations: %w", err)
	}
	plan, err := readJSON[map[string]any](cfg.plan)
	if err != nil {
		return fmt.Errorf("read plan: %w", err)
	}
	if err := terraform.MergeLocationsIntoPlan(locs, plan); err != nil {
		return fmt.Errorf("merge locations into plan: %w", err)
	}
	by, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	if err := writeContents(cfg.out, by); err != nil {
		return fmt.Errorf("write plan: %w", err)
	}

	return nil
}

func readJSON[T any](path string) (T, error) {
	var t T
	by, err := readContents(path)
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal(by, &t); err != nil {
		return t, err
	}
	return t, nil
}

func readContents(path string) ([]byte, error) {
	if stdinout(path) {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func writeContents(path string, data []byte) error {
	if stdinout(path) {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, os.ModePerm)
}

func stdinout(path string) bool {
	return path == " " || path == "-"
}
