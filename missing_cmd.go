package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/google/subcommands"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type missingCmd struct {
}

func (*missingCmd) Name() string     { return "missing" }
func (*missingCmd) Synopsis() string { return "list missing pages" }
func (*missingCmd) Usage() string {
	return `missing:
  Listing pages with links to missing pages.
`
}

func (cmd *missingCmd) SetFlags(f *flag.FlagSet) {
}

func (cmd *missingCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	return missingCli(os.Stdout, f.Args())
}

func missingCli(w io.Writer, args []string) subcommands.ExitStatus {
	names := make(map[string]bool)
	err := filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		filename := path
		if info.IsDir() || strings.HasPrefix(filepath.Base(filename), ".") {
			return nil
		}
		if strings.HasSuffix(filename, ".md") {
			name := strings.TrimSuffix(filename, ".md")
			names[name] = true
		} else {
			names[filename] = false
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(w, err)
		return subcommands.ExitFailure
	}
	found := false
	for name, isPage := range names {
		if !isPage {
			continue
		}
		p, err := loadPage(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Loading %s: %s\n", name, err)
			return subcommands.ExitFailure
		}
		for _, link := range p.links() {
			u, err := url.Parse(link)
			if err != nil {
				fmt.Fprintln(os.Stderr, name, err)
				return subcommands.ExitFailure
			}
			if u.Scheme == "" && u.Path != "" && !strings.HasPrefix(u.Path, "/") {
				// feeds can work if the matching page works
				u.Path = strings.TrimSuffix(u.Path, ".rss")
				// links to the source file can work
				u.Path = strings.TrimSuffix(u.Path, ".md")
				// pages containing a colon need the ./ prefix
				u.Path = strings.TrimPrefix(u.Path, "./")
				// check whether the destinatino is a known page
				destination, err := url.PathUnescape(u.Path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Cannot decode %s: %s\n", link, err)
					return subcommands.ExitFailure
				}
				_, ok := names[destination]
				if !ok {
					if !found {
						fmt.Fprintln(w, "Page\tMissing")
						found = true
					}
					fmt.Fprintf(w, "%s\t%s\n", name, link)
				}
			}
		}
	}
	if !found {
		fmt.Fprintln(w, "No missing pages found.")
	}
	return subcommands.ExitSuccess
}

// links parses the page content and returns an array of link destinations.
func (p *Page) links() []string {
	var links []string
	parser, _ := wikiParser()
	doc := markdown.Parse(p.Body, parser)
	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		if entering {
			switch v := node.(type) {
			case *ast.Link:
				links = append(links, string(v.Destination))
			}
		}
		return ast.GoToNext
	})
	return links
}
