package pager

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	tea "github.com/charmbracelet/bubbletea"
)

func newPaginatorModel(content string) Paginator {
	items := strings.Split(content, "\n")

	_, height, err := term.GetSize(int(os.Stdout.Fd()))
	perPage := 10
	if err == nil {
		perPage = height - 4
		if perPage < 1 {
			perPage = 1
		}
	}

	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = perPage
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
	p.SetTotalPages(len(items))

	return Paginator{
		paginator: p,
		items:     items,
	}
}

type Paginator struct {
	items     []string
	paginator paginator.Model
}

func (m Paginator) Init() tea.Cmd {
	return nil
}

func (m Paginator) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	m.paginator, cmd = m.paginator.Update(msg)
	return m, cmd
}

func (m Paginator) View() string {
	var b strings.Builder
	b.WriteString("\n")
	start, end := m.paginator.GetSliceBounds(len(m.items))
	total := len(m.items)
	width := len(fmt.Sprintf("%d", total))
	for i, item := range m.items[start:end] {
		num := start + i + 1
		b.WriteString(fmt.Sprintf("%*d %s\n", width, num, item))
	}
	b.WriteString(fmt.Sprintf("%s", m.paginator.View()))
	b.WriteString(fmt.Sprintf("\n← %d/%d → navigate • esc quit\n", m.paginator.Page+1, m.paginator.TotalPages))
	return b.String()
}

func (o *Options) Paginate() error {
	if o.Content == "" {
		return nil
	}
	p := tea.NewProgram(newPaginatorModel(o.Content), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
