package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/subcommands"
	"io"
	"os"
)

type notifyCmd struct {
}

func (*notifyCmd) Name() string     { return "notify" }
func (*notifyCmd) Synopsis() string { return "add links to changes.md, index.md, and hashtag pages" }
func (*notifyCmd) Usage() string {
	return `notify <page name> ...:
  For each page, add entries to changes.md, index.md, and hashtag pages.
  This is useful when writing pages offline and replicates the behaviour
  triggered by the "Add link to the list of changes" checkbox, online.
`
}

func (cmd *notifyCmd) SetFlags(f *flag.FlagSet) {
}

func (cmd *notifyCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	return notifyCli(os.Stdout, f.Args())
}

func notifyCli(w io.Writer, args []string) subcommands.ExitStatus {
	index.load()
	for _, name := range args {
		p, err := loadPage(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Loading %s: %s\n", name, err)
			return subcommands.ExitFailure
		}
		err = p.notify()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", name, err)
			return subcommands.ExitFailure
		}
	}
	return subcommands.ExitSuccess
}
