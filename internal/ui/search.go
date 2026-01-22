package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"dab-downloader/internal/models"
)

// Searcher interface to break import cycle
type Searcher interface {
	Search(ctx context.Context, query string, searchType string, limit int, debug bool) (*models.SearchResults, error)
}

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
	selected    bool
}

func (i searchItem) Title() string {
	if i.selected {
		return "[x] " + i.title
	}
	return "[ ] " + i.title
}
func (i searchItem) Description() string { return i.desc }
func (i searchItem) FilterValue() string { return i.title }

type model struct {
	list          list.Model
	selectedItems []interface{}
	itemTypes     []string
	quitting      bool
	multiSelect   bool // Track if user has engaged in multi-selection
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

		case " ": // Space to toggle selection
			idx := m.list.Index()
			if idx >= 0 && idx < len(m.list.Items()) {
				item := m.list.Items()[idx].(searchItem)
				item.selected = !item.selected
				m.list.SetItem(idx, item)
				m.multiSelect = true
			}
			return m, nil

		case "enter":
			// If user used multi-select (space), collect all selected items
			if m.multiSelect {
				for _, it := range m.list.Items() {
					item := it.(searchItem)
					if item.selected {
						m.selectedItems = append(m.selectedItems, item.originalItem)
						m.itemTypes = append(m.itemTypes, item.itemType)
					}
				}
				// If nothing was selected despite toggling (e.g. unselected everything), fallback to current
				if len(m.selectedItems) == 0 {
					i, ok := m.list.SelectedItem().(searchItem)
					if ok {
						m.selectedItems = append(m.selectedItems, i.originalItem)
						m.itemTypes = append(m.itemTypes, i.itemType)
					}
				}
			} else {
				// Single selection mode
				i, ok := m.list.SelectedItem().(searchItem)
				if ok {
					m.selectedItems = append(m.selectedItems, i.originalItem)
					m.itemTypes = append(m.itemTypes, i.itemType)
				}
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return quitTextStyle.Render("Search cancelled.")
	}
	return "\n" + m.list.View()
}

// HandleSearch performs the search and UI interaction
func HandleSearch(ctx context.Context, api Searcher, query string, searchType string, debug bool, auto bool) ([]interface{}, []string, error) {
	Info.Printf("🔎 Searching for '%s' (type: %s)...\n", query, searchType)

	results, err := api.Search(ctx, query, searchType, 20, debug) // Increased limit for better list
	if err != nil {
		return nil, nil, err
	}

	totalResults := len(results.Artists) + len(results.Albums) + len(results.Tracks)
	if totalResults == 0 {
		Warning.Println("No results found.")
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
	l.Title = "Search Results (Space to select, Enter to confirm)"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, nil, fmt.Errorf("error running selection UI: %w", err)
	}

	finalM, ok := finalModel.(model)
	if !ok {
		return nil, nil, fmt.Errorf("could not cast model")
	}

	if finalM.quitting {
		return nil, nil, nil // User quit
	}

	return finalM.selectedItems, finalM.itemTypes, nil
}