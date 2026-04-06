package main

import (
	"fmt"
	"os"
	"strings"
)

const helpText = `Git Worktree Manager

Usage:
  wt <branch>           Smart switch/create (prompts before creating)
  wt -y <branch>        Skip prompts
  wt -q <branch>        Quiet mode (only output path)
  wt -n <branch>        No tmux (skip window creation/switching)
  wt -yqn <branch>      Combine flags (for Claude/scripts)
  wt z [query]          Fuzzy find worktree, output path (cd "$(wt z)")
  wt main               Switch to root repository window
  wt list               List all worktrees
  wt remove <branch>    Remove worktree + kill window
  wt clean              Remove stale worktrees (merged, squash-merged, deleted)
  wt clean -i           Interactive explorer: inspect worktrees, force-remove
  wt help               Show this help

Model: Session = Project, Window = Worktree

Smart mode:
  Worktree exists     → switch to window (unless -n)
  Branch exists       → prompt to create worktree
  Branch not found    → prompt to create new branch

Worktree location: .worktrees/<branch-name>`

type flags struct {
	yes         bool
	quiet       bool
	noSwitch    bool
	interactive bool
}

func parseArgs(args []string) (flags, []string) {
	var f flags
	var rest []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && arg != "-" {
			chars := arg[1:]
			allShort := true
			for _, c := range chars {
				switch c {
				case 'y':
					f.yes = true
				case 'q':
					f.quiet = true
				case 'n':
					f.noSwitch = true
				case 'i':
					f.interactive = true
				default:
					allShort = false
				}
			}
			if !allShort {
				rest = append(rest, arg)
			}
			continue
		}

		switch arg {
		case "--yes":
			f.yes = true
		case "--quiet":
			f.quiet = true
		case "--no-switch":
			f.noSwitch = true
		case "--interactive":
			f.interactive = true
		default:
			rest = append(rest, arg)
		}
	}

	return f, rest
}

func main() {
	f, args := parseArgs(os.Args[1:])
	_ = f

	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}

	switch sub {
	case "help", "-h", "--help", "":
		fmt.Println(helpText)
	case "list", "ls":
		fmt.Fprintln(os.Stderr, "not implemented: list")
		os.Exit(1)
	case "remove", "rm":
		fmt.Fprintln(os.Stderr, "not implemented: remove")
		os.Exit(1)
	case "clean", "prune":
		fmt.Fprintln(os.Stderr, "not implemented: clean")
		os.Exit(1)
	case "z":
		fmt.Fprintln(os.Stderr, "not implemented: z")
		os.Exit(1)
	case "main":
		fmt.Fprintln(os.Stderr, "not implemented: main")
		os.Exit(1)
	default:
		// Smart mode: treat as branch name
		fmt.Fprintln(os.Stderr, "not implemented: smart")
		os.Exit(1)
	}
}
