package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/zefhemel/matterless/pkg/definition"
	"os"
	"strings"
)

func prerenderCommand() *cobra.Command {
	var (
		dataDir string
		watch   bool
	)
	var cmd = &cobra.Command{
		Use:   "prerender file.md",
		Short: "Prerenders (includes all imports, expands macros) for a file",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path := args[0]
			prerenderFile(path, dataDir)
			if watch {
				watchFiles([]string{path}, func(path string) {
					fmt.Printf("File %s changed on disk, rerendering...", path)
					prerenderFile(path, dataDir)
				})
				busyLoop()
			}
		},
	}
	cmd.Flags().StringVar(&dataDir, "data", "./mls-data", "Path to keep Matterless state")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch file for changes and rerender automatically")

	return cmd
}

func prerenderFile(path string, dataDir string) {
	buf, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	defs, err := definition.Parse(string(buf))
	if err != nil {
		log.Fatal(err)
	}
	if err := defs.InlineImports(fmt.Sprintf("%s/.importcache", dataDir)); err != nil {
		log.Fatal(err)
	}
	if err := defs.ExpandMacros(); err != nil {
		log.Fatal(err)
	}
	outPath := strings.Replace(path, ".md", ".rendered.md", 1)
	if err := os.WriteFile(outPath, []byte(defs.Markdown()), 0600); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Output in ", outPath)
}
