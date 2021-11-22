package main

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewPollOverlay(app *App) *PollOverlay {
	p := &PollOverlay{
		app:        app,
		Flex:       tview.NewFlex(),
		TextTop:    tview.NewTextView(),
		TextBottom: tview.NewTextView(),
		List:       tview.NewList(),
	}

	p.TextTop.SetBackgroundColor(app.Config.Style.Background)
	p.TextTop.SetTextColor(app.Config.Style.Text)
	p.TextTop.SetDynamicColors(true)
	p.TextBottom.SetBackgroundColor(app.Config.Style.Background)
	p.TextBottom.SetDynamicColors(true)
	p.List.SetBackgroundColor(app.Config.Style.Background)
	p.List.SetMainTextColor(app.Config.Style.Text)
	p.List.SetSelectedBackgroundColor(app.Config.Style.ListSelectedBackground)
	p.List.SetSelectedTextColor(app.Config.Style.ListSelectedText)
	p.List.ShowSecondaryText(false)
	p.List.SetHighlightFullLine(true)
	p.Flex.SetDrawFunc(app.Config.ClearContent)
	var items []string
	items = append(items, ColorKey(app.Config, "", "A", "dd"))
	items = append(items, ColorKey(app.Config, "", "D", "elete"))
	items = append(items, ColorKey(app.Config, "", "E", "xpiration"))
	items = append(items, ColorKey(app.Config, "Toogle ", "M", "multiple"))
	items = append(items, ColorKey(app.Config, "Toogle ", "H", "ide total"))
	p.TextBottom.SetText(strings.Join(items, " "))
	return p
}

type PollOverlay struct {
	app        *App
	Flex       *tview.Flex
	TextTop    *tview.TextView
	TextBottom *tview.TextView
	List       *tview.List
	options    []string
	expiration int
	multiple   bool
	hideTotal  bool
}

func (p *PollOverlay) NewPoll() {
	p.List.Clear()
	p.TextTop.SetText("")
}

func (p *PollOverlay) Prev() {
	index := p.List.GetCurrentItem()
	if index-1 >= 0 {
		p.List.SetCurrentItem(index - 1)
	}
}

func (p *PollOverlay) Next() {
	index := p.List.GetCurrentItem()
	if index+1 < p.List.GetItemCount() {
		p.List.SetCurrentItem(index + 1)
	}
}

func (p *PollOverlay) ToggleMultiple() {
	p.multiple = !p.multiple
}
func (p *PollOverlay) ToggleHideTotal() {
	p.hideTotal = !p.hideTotal
}

func (v *PollOverlay) InputHandler(event *tcell.EventKey) {
	if event.Key() == tcell.KeyRune {
		switch event.Rune() {
		case 'j', 'J':
			v.Next()
		case 'k', 'K':
			v.Prev()
		case 'm', 'M':
			v.ToggleMultiple()
		case 'h', 'H':
			v.ToggleHideTotal()
		case 'q', 'Q':
			v.app.UI.SetFocus(MessageFocus)
		}
	} else {
		switch event.Key() {
		case tcell.KeyUp:
			v.Prev()
		case tcell.KeyDown:
			v.Next()
		case tcell.KeyEsc:
			v.app.UI.SetFocus(MessageFocus)
		}
	}
}
