package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/blackhawk42/doppelrm/pkg/doppelparser"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// keyMap is the keybindings of the Bubbletea program
type keyMap struct {
	// Move one file up
	up key.Binding

	// Move one file down
	down key.Binding

	// Move to the previous hash
	left key.Binding

	// Move to the next hash
	right key.Binding

	// Toggle whether the current file is selected
	toggle key.Binding

	// Exit the program while confirming choices
	enter key.Binding

	// Quit without doing anything
	quit key.Binding

	// Toggle long and short help
	help key.Binding
}

// defaultKeyMap delivers the default keybindings of the program
func defaultKeyMap() *keyMap {
	return &keyMap{
		up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "previous hash"),
		),
		right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next hash"),
		),
		toggle: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle choice"),
		),
		enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "confirm choices"),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/CTRL+C", "quit without making changes"),
		),
		help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle full help"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.toggle, k.quit, k.enter, k.help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.toggle, k.quit, k.enter, k.help},
		{k.up, k.down, k.left, k.right},
	}
}

// filename represents a single filename
type filename struct {
	name     string
	selected bool
	valid    bool
}

// filesChoice represents a choice of filenames
type filesChoice struct {
	files  []*filename
	cursor int
}

func (fc *filesChoice) Next() {
	fc.cursor = (fc.cursor + 1) % len(fc.files)
}

func (fc *filesChoice) Prev() {
	fc.cursor--
	if fc.cursor < 0 {
		fc.cursor = len(fc.files) - 1
	}
}

func (fc *filesChoice) CurrentFile() *filename {
	return fc.files[fc.cursor]
}

// collision represents a single collision with a hash and multiple fileChoices
type collision struct {
	hash        string
	fileChoices filesChoice
}

// collisionChoice represents a choice between multiple collisions
type collisionChoice struct {
	cursor     int
	collisions []*collision
}

func (cc *collisionChoice) Next() {
	cc.cursor = (cc.cursor + 1) % len(cc.collisions)
}

func (cc *collisionChoice) Prev() {
	cc.cursor--
	if cc.cursor < 0 {
		cc.cursor = len(cc.collisions) - 1
	}
}

func (cc *collisionChoice) CurrentCollision() *collision {
	return cc.collisions[cc.cursor]
}

// utility to push a new error into finalErr, without repeats
func pushError(err error, s []string) []string {
	msg := err.Error()
	for _, oldMsg := range s {
		if msg == oldMsg {
			return s
		}
	}

	return append(s, msg)
}

// bblModel is the Bubbletea model of the application
type bblModel struct {
	// Whether there was an error on the executuin of the program
	finalErr error

	// Temporary erros, just for show during program execution
	tempErrors []string

	// Whether choices were confirmed before exiting
	confirmedChoices bool

	// The keybindings of the application.
	keymap *keyMap

	// All the collisions to select from
	collisionChoices *collisionChoice

	// The help footer
	help help.Model

	// The progress bar
	progress progress.Model

	// Style for setting all the application's width width
	widthStyle lipgloss.Style

	// Style for normal text
	normalStyle lipgloss.Style

	// Style for bold text
	boldStyle lipgloss.Style

	// Style for hashes
	hashStyle lipgloss.Style

	// Style for the filenames, including checkbox
	filenameStyle lipgloss.Style

	// Style for files considered invalid, including checkbox
	invalidStyle lipgloss.Style

	// Style for the little progress count over the progress bar
	progressCountStyle lipgloss.Style

	// Style for temporary errors
	tempErrorsStyle lipgloss.Style
}

func newBblModel(dc *doppelparser.DoppelCollisions) bblModel {
	m := bblModel{
		finalErr:         nil,
		confirmedChoices: false,
		tempErrors:       nil,
		keymap:           defaultKeyMap(),
		collisionChoices: &collisionChoice{
			cursor:     0,
			collisions: make([]*collision, 0, dc.Len()),
		},
		help:               help.New(),
		widthStyle:         lipgloss.NewStyle(),
		progress:           progress.New(progress.WithoutPercentage()),
		normalStyle:        lipgloss.NewStyle(),
		boldStyle:          lipgloss.NewStyle().Bold(true),
		hashStyle:          lipgloss.NewStyle().PaddingLeft(2),
		filenameStyle:      lipgloss.NewStyle(),
		progressCountStyle: lipgloss.NewStyle().Faint(true),
		invalidStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		tempErrorsStyle:    lipgloss.NewStyle().Faint(true),
	}

	for hash, cols := range dc.Iter() {
		collisionChoice := &collision{
			hash: hash,
			fileChoices: filesChoice{
				cursor: 0,
				files:  make([]*filename, 0, len(cols)),
			},
		}

		for _, c := range cols {
			fn := &filename{
				name:     c,
				selected: true,
				valid:    true,
			}

			collisionChoice.fileChoices.files = append(collisionChoice.fileChoices.files, fn)
		}

		m.collisionChoices.collisions = append(m.collisionChoices.collisions, collisionChoice)
	}

	return m
}

func (m bblModel) Init() tea.Cmd {
	return CheckFilesExistsCmd(m.collisionChoices.CurrentCollision().fileChoices.files)
}

// fileExistsResponse is a response message to a CheckFilesExistsCmd command.
type fileExistsResponse struct {
	// Any error that may have occurred
	err error

	// All the files that were found to exist
	exists []*filename

	// All the files that were found to not exist
	not_exists []*filename
}

// CheckFilesExistsCmd is a command to check if the given files exist
//
// Final command returns a fileExistsResponse message. Note that this will only return
// the message, *not* modify the files. This should be handled by the model Update
// method.
func CheckFilesExistsCmd(files []*filename) tea.Cmd {
	return func() tea.Msg {
		response := fileExistsResponse{}

		for _, f := range files {
			_, err := os.Stat(f.name)
			if err == nil {
				response.exists = append(response.exists, f)
			} else if errors.Is(err, os.ErrNotExist) {
				response.not_exists = append(response.not_exists, f)
			} else {
				response.err = err
				break
			}
		}

		return tea.Msg(response)
	}
}

func (m bblModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width
		m.progress.Width = msg.Width
		// Styles
		m.widthStyle = m.widthStyle.Width(msg.Width)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.down):
			m.collisionChoices.CurrentCollision().fileChoices.Next()
		case key.Matches(msg, m.keymap.up):
			m.collisionChoices.CurrentCollision().fileChoices.Prev()
		case key.Matches(msg, m.keymap.right):
			m.collisionChoices.Next()
			cmds = append(cmds, CheckFilesExistsCmd(m.collisionChoices.CurrentCollision().fileChoices.files))
		case key.Matches(msg, m.keymap.left):
			m.collisionChoices.Prev()
			cmds = append(cmds, CheckFilesExistsCmd(m.collisionChoices.CurrentCollision().fileChoices.files))
		case key.Matches(msg, m.keymap.toggle):
			m.collisionChoices.CurrentCollision().fileChoices.CurrentFile().selected = !m.collisionChoices.CurrentCollision().fileChoices.CurrentFile().selected
		case key.Matches(msg, m.keymap.help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.enter):
			m.confirmedChoices = true
			return m, tea.Quit
		}
	case fileExistsResponse:
		if msg.err != nil {
			m.tempErrors = pushError(msg.err, m.tempErrors)
		}

		for _, f := range msg.exists {
			f.valid = true
		}

		for _, f := range msg.not_exists {
			f.valid = false
		}
	}

	return m, tea.Batch(cmds...)
}

func (m bblModel) View() string {
	var result strings.Builder

	result.WriteString("Select all you want to ")
	result.WriteString(m.boldStyle.Render("stay"))
	result.WriteString("\n\n")

	result.WriteString(m.hashStyle.Render(m.collisionChoices.CurrentCollision().hash))
	result.WriteString("\n\n")

	for i, file := range m.collisionChoices.CurrentCollision().fileChoices.files {
		currentSymbol := " "
		if i == m.collisionChoices.CurrentCollision().fileChoices.cursor {
			currentSymbol = ">"
		}

		selectedSymbol := " "
		if file.selected {
			selectedSymbol = "X"
		}

		fileLine := fmt.Sprintf("%s [%s] %s", currentSymbol, selectedSymbol, file.name)
		fileSytle := m.filenameStyle
		if !file.valid {
			fileSytle = m.invalidStyle
		}
		result.WriteString(
			fileSytle.Render(fileLine),
		)
		result.WriteString("\n")
	}

	result.WriteString("\n")
	result.WriteString(
		m.progressCountStyle.Render(fmt.Sprintf("%d/%d", m.collisionChoices.cursor+1, len(m.collisionChoices.collisions))),
	)
	result.WriteString("\n")
	result.WriteString(m.progress.ViewAs(
		float64(m.collisionChoices.cursor+1) / float64(len(m.collisionChoices.collisions))),
	)

	result.WriteString("\n")
	result.WriteString(m.help.View(m.keymap))

	if len(m.tempErrors) > 0 {
		result.WriteString("\n")
		for _, err := range m.tempErrors {
			result.WriteString(m.tempErrorsStyle.Render(err))
			result.WriteString("\n")
		}
	}

	return m.widthStyle.Render(result.String())
}
