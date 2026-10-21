package main

import (
	"fmt"
	"io"
	"strings"
)

type CLI struct {
	Stdout io.Writer
	Stderr io.Writer
}

func NewCLI(stdout io.Writer, stderr io.Writer) CLI {
	return CLI{Stdout: stdout, Stderr: stderr}
}

func (c CLI) Run(args []string) int {
	if len(args) == 0 {
		c.usage()
		return 1
	}
	switch args[0] {
	case "--help", "-h", "help":
		c.usage()
		return 0
	case "--list", "list":
		for _, name := range AvailableScenarios() {
			fmt.Fprintln(c.Stdout, name)
		}
		return 0
	case "scenario", "run":
		if len(args) < 2 {
			fmt.Fprintln(c.Stderr, "scenario name required")
			return 1
		}
		run, err := RunScenario(args[1])
		if err != nil {
			fmt.Fprintln(c.Stderr, err.Error())
			return 1
		}
		report := BuildReport(run)
		if err := WriteJSON(c.Stdout, report); err != nil {
			fmt.Fprintln(c.Stderr, err.Error())
			return 1
		}
		return 0
	case "status":
		if len(args) < 2 {
			fmt.Fprintln(c.Stderr, "scenario name required")
			return 1
		}
		run, err := RunScenario(args[1])
		if err != nil {
			fmt.Fprintln(c.Stderr, err.Error())
			return 1
		}
		if err := WriteJSON(c.Stdout, BuildStatusReport(run)); err != nil {
			fmt.Fprintln(c.Stderr, err.Error())
			return 1
		}
		return 0
	case "validate":
		if len(args) < 2 {
			fmt.Fprintln(c.Stderr, "scenario name required")
			return 1
		}
		run, err := RunScenario(args[1])
		if err != nil {
			fmt.Fprintln(c.Stderr, err.Error())
			return 1
		}
		report := BuildReport(run)
		if !ValidateReport(report) {
			fmt.Fprintln(c.Stderr, "scenario contract failed")
			return 1
		}
		fmt.Fprintf(c.Stdout, "ok %s %s\n", report.Scenario, report.StateDigest)
		return 0
	default:
		fmt.Fprintf(c.Stderr, "unknown command %q\n", args[0])
		return 1
	}
}

func (c CLI) usage() {
	lines := []string{
		"riftdtl commands:",
		"  --list",
		"  scenario <name>",
		"  status <name>",
		"  validate <name>",
	}
	fmt.Fprintln(c.Stdout, strings.Join(lines, "\n"))
}
