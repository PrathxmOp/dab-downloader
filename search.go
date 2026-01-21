package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type searchItem struct {
	title       string
	desc        string
	originalItem interface{} // Store the original Artist, Album, or Track object
	itemType    string      // "artist", "album", or "track"
}

func (i searchItem) Title() string       { return i.title }
func (i searchItem) Description() string { return i.desc }
func (i searchItem) FilterValue() string { return i.title }

type model struct {
	list          list.Model
	selectedItems []interface{}
	itemTypes     []string
	choice        *searchItem
	quitting      bool
	err           error
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(searchItem)
			if ok {
				m.choice = &i
				m.selectedItems = append(m.selectedItems, i.originalItem)
				m.itemTypes = append(m.itemTypes, i.itemType)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.choice != nil {
		return quitTextStyle.Render(fmt.Sprintf("Downloading %s...", m.choice.title))
	}
	if m.quitting {
		return quitTextStyle.Render("Search cancelled.")
	}
	return "\n" + m.list.View()
}

func handleSearch(ctx context.Context, api *DabAPI, query string, searchType string, debug bool, auto bool) ([]interface{}, []string, error) {
	colorInfo.Printf("🔎 Searching for '%s' (type: %s)...\n", query, searchType)

	results, err := api.Search(ctx, query, searchType, 20, debug) // Increased limit for better list
	if err != nil {
		return nil, nil, err
	}

	totalResults := len(results.Artists) + len(results.Albums) + len(results.Tracks)
	if totalResults == 0 {
		colorWarning.Println("No results found.")
		return nil, nil, nil
	}

	if auto {
		var selectedItems []interface{}
		var itemTypes []string
		if len(results.Artists) > 0 {
			selectedItems = append(selectedItems, results.Artists[0])
			itemTypes = append(itemTypes, "artist")
		} else if len(results.Albums) > 0 {
			selectedItems = append(selectedItems, results.Albums[0])
			itemTypes = append(itemTypes, "album")
		} else if len(results.Tracks) > 0 {
			selectedItems = append(selectedItems, results.Tracks[0])
			itemTypes = append(itemTypes, "track")
		}
		return selectedItems, itemTypes, nil
	}

	var items []list.Item

	for _, artist := range results.Artists {
		items = append(items, searchItem{
			title:        artist.Name,
			desc:         "Artist",
			originalItem: artist,
			itemType:     "artist",
		})
	}
	for _, album := range results.Albums {
		items = append(items, searchItem{
			title:        album.Title,
			desc:         fmt.Sprintf("Album by %s", album.Artist),
			originalItem: album,
			itemType:     "album",
		})
	}
	for _, track := range results.Tracks {
		items = append(items, searchItem{
			title:        track.Title,
			desc:         fmt.Sprintf("Track by %s (%s)", track.Artist, track.Album),
			originalItem: track,
			itemType:     "track",
		})
	}

	const defaultWidth = 20
	const listHeight = 14

	l := list.New(items, list.NewDefaultDelegate(), defaultWidth, listHeight)
	l.Title = "Search Results"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	// Because we run the program and it updates the model copy (value receiver in Update if not pointer),
	// wait, tea.NewProgram returns the *final* model.
	// We need to capture the return value.
	
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, nil, fmt.Errorf("error running selection UI: %w", err)
	}

	finalM, ok := finalModel.(model)
	if !ok {
		return nil, nil, fmt.Errorf("could not cast model")
	}

	if finalM.quitting && finalM.choice == nil {
		return nil, nil, nil // User quit
	}

	return finalM.selectedItems, finalM.itemTypes, nil
}