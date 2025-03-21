package main

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/blackhawk42/doppelrm/pkg/doppelparser"
	tea "github.com/charmbracelet/bubbletea"
)

type Cli struct {
	File string `arg:"" type:"existingfile" help:"The doppel output file. \"-\" means stdin."`
}

func main() {
	var cli Cli
	kongCtx := kong.Parse(
		&cli,
		kong.Description("Interactively delete files from a doppel output"),
	)

	// Parse doppel file
	var input io.ReadCloser
	if cli.File == "-" {
		input = io.NopCloser(os.Stdin)
	} else {
		f, err := os.Open(cli.File)
		kongCtx.FatalIfErrorf(err, fmt.Sprintf("while opening file %s: %v", cli.File, err))

		input = f
	}

	doppelFile, err := doppelparser.ParseDoppelFile(input)
	kongCtx.FatalIfErrorf(err, fmt.Sprintf("while reading input file %s: %v", cli.File, err))
	input.Close()

	// Setup and start Bubbletea
	model := newBblModel(doppelFile)
	bblProgram := tea.NewProgram(model)
	m, err := bblProgram.Run()
	kongCtx.FatalIfErrorf(err, fmt.Sprintf("while starting Bubbletea program: %v", err))
	model = m.(bblModel)
	kongCtx.FatalIfErrorf(model.finalErr, fmt.Sprintf("%v", model.finalErr))

	// If terminated without confirming choices, just quit and do nothing
	if !model.confirmedChoices {
		return
	}

	// Delete all files that weren't selected
	for _, col := range model.collisionChoices.collisions {
		for _, file := range col.fileChoices.files {
			if !file.selected {
				err = os.Remove(file.name)
				if err == nil {
					fmt.Fprintf(os.Stderr, "removed %s\n", file.name)
				} else {
					fmt.Fprintf(os.Stderr, "error while removing %s: %v\n", file.name, err)
				}
			} else {
				fmt.Fprintf(os.Stderr, "kept %s\n", file.name)
			}
		}
	}
}
