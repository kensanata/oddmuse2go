package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/google/subcommands"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type staticCmd struct {
}

func (*staticCmd) Name() string     { return "static" }
func (*staticCmd) Synopsis() string { return "Render site into static HTML files." }
func (*staticCmd) Usage() string {
	return `static <dir name>:
  Create static copies in the given directory.
`
}

func (cmd *staticCmd) SetFlags(f *flag.FlagSet) {
}

func (cmd *staticCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	args := f.Args()
	if len(args) != 1 {
		fmt.Println("Exactly one target directory is required")
		return subcommands.ExitFailure
	}
	return staticCli(filepath.Clean(args[0]))
}

func staticCli(dir string) subcommands.ExitStatus {
	err := os.Mkdir(dir, 0755)
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}
	initAccounts()
	err = filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		return staticFile(path, dir, info, err)
	})
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

// staticFile is used to walk the file trees and do the right thing for the destination directory: create
// subdirectories, link files, render HTML files.
func staticFile(path, dir string, info fs.FileInfo, err error) error {
	if err != nil {
		return err
	}
	filename := path
	// skip "hidden" files and backup files, avoid recursion
	if strings.HasPrefix(filename, ".") ||
		strings.HasSuffix(filename, "~") ||
		strings.HasPrefix(filename, dir) {
		return nil
	}
	// recreate subdirectories
	if info.IsDir() {
		return os.Mkdir(filepath.Join(dir, filename), 0755)
	}
	// render pages
	if strings.HasSuffix(filename, ".md") {
		return staticPage(filename, dir)
	}
	// remaining files are linked
	return os.Link(filename, filepath.Join(dir, filename))
}

// staticPage takes the filename of a page (ending in ".md") and generates a static HTML page.
func staticPage(filename, dir string) error {
	name := strings.TrimSuffix(filename, ".md")
	p, err := loadPage(name)
	if err != nil {
		fmt.Printf("Cannot load %s: %s\n", name, err)
		return err
	}
	p.handleTitle(true)
	// instead of p.renderHtml() we do it all ourselves, appending ".html" to all the local links
	parser, hashtags := wikiParser()
	doc := markdown.Parse(p.Body, parser)
	ast.WalkFunc(doc, staticLinks)
	opts := html.RendererOptions{
		Flags: html.CommonFlags,
	}
	renderer := html.NewRenderer(opts)
	maybeUnsafeHTML := markdown.Render(doc, renderer)
	p.Name = nameEscape(p.Name)
	p.Html = sanitizeBytes(maybeUnsafeHTML)
	p.Language = language(p.plainText())
	p.Hashtags = *hashtags
	return p.write(filepath.Join(dir, name+".html"))
}

// staticLinks checks a node and if it is a link to a local page, it appends ".html" to the link destination.
func staticLinks(node ast.Node, entering bool) ast.WalkStatus {
	if entering {
		switch v := node.(type) {
		case *ast.Link:
			// not an absolute URL, not a full URL, not a mailto: URI
			if !bytes.HasPrefix(v.Destination, []byte("/")) &&
				!bytes.Contains(v.Destination, []byte("://")) &&
				!bytes.HasPrefix(v.Destination, []byte("mailto:")) {
				// pointing to a page file (instead of an image file, for example).
				fn, err := url.PathUnescape(string(v.Destination))
				if err != nil {
					return ast.GoToNext
				}
				_, err = os.Stat(fn + ".md")
				if err != nil {
					return ast.GoToNext
				}
				v.Destination = append(v.Destination, []byte(".html")...)
			}
		}
	}
	return ast.GoToNext
}

func (p *Page) write(destination string) error {
	t := "static.html"
	f, err := os.Create(destination)
	if err != nil {
		fmt.Printf("Cannot create %s.html: %s\n", destination, err)
		return err
	}
	err = templates.ExecuteTemplate(f, t, p)
	if err != nil {
		fmt.Printf("Cannot execute %s template for %s: %s\n", t, destination, err)
		return err
	}
	return nil
}