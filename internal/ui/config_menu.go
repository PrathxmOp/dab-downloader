package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"dab-downloader/internal/config"
)

var (
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle         = focusedStyle.Copy()
	noStyle             = lipgloss.NewStyle()
	helpStyleConfig     = blurredStyle.Copy()
	cursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	focusedButton = focusedStyle.Copy().Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
)

type ConfigModel struct {
	focusIndex int
	inputs     []textinput.Model
	conf       *config.Config
	quitting   bool
	saved      bool
}

func NewConfigModel(conf *config.Config) ConfigModel {
	m := ConfigModel{
		inputs: make([]textinput.Model, 6),
		conf:   conf,
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 256

		switch i {
		case 0:
			t.Placeholder = "API URL"
			t.SetValue(conf.APIURL)
			t.Prompt = "API URL: "
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "Download Location"
			t.SetValue(conf.DownloadLocation)
			t.Prompt = "Download Path: "
		case 2:
			t.Placeholder = "5"
			t.SetValue(strconv.Itoa(conf.Parallelism))
			t.Prompt = "Parallelism: "
			t.CharLimit = 3
		case 3:
			t.Placeholder = "flac"
			t.SetValue(conf.Format)
			t.Prompt = "Format: "
			t.CharLimit = 5
		case 4:
			t.Placeholder = "320"
			t.SetValue(conf.Bitrate)
			t.Prompt = "Bitrate: "
			t.CharLimit = 4
		case 5:
			t.Placeholder = "true"
			t.SetValue(strconv.FormatBool(conf.VerifyDownloads))
			t.Prompt = "Verify Downloads (true/false): "
			t.CharLimit = 5
		}

		m.inputs[i] = t
	}

	return m
}

func (m ConfigModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			if s == "enter" && m.focusIndex == len(m.inputs) {
				// Submit
				m.saved = true
				m.updateConfig()
				return m, tea.Quit
			}

			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					// Set focused state
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *ConfigModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m ConfigModel) View() string {
	if m.quitting {
		return "Configuration cancelled.\n"
	}
	if m.saved {
		return "Configuration saved!\n"
	}

	var b strings.Builder

	b.WriteString("\n  📝 Configuration Editor\n\n")

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	b.WriteString(helpStyleConfig.Render("cursor mode is "))
	b.WriteString(cursorModeHelpStyle.Render(m.inputs[0].Cursor.Mode().String()))
	b.WriteString(helpStyleConfig.Render(" (ctrl+r to change style)"))

	return b.String()
}

func (m *ConfigModel) updateConfig() {
	m.conf.APIURL = m.inputs[0].Value()
	m.conf.DownloadLocation = m.inputs[1].Value()
	
	if p, err := strconv.Atoi(m.inputs[2].Value()); err == nil {
		m.conf.Parallelism = p
	}
	
m.conf.Format = m.inputs[3].Value()
m.conf.Bitrate = m.inputs[4].Value()
	
	if v, err := strconv.ParseBool(m.inputs[5].Value()); err == nil {
		m.conf.VerifyDownloads = v
	}
}

// RunConfigMenu runs the configuration menu
func RunConfigMenu(conf *config.Config) error {
	p := tea.NewProgram(NewConfigModel(conf))
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	
m := finalModel.(ConfigModel)
	if m.saved {
		return config.SaveConfig(config.GetConfigFilePath(), conf)
	}
	return nil
}
