package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewExpirationOverlay(app *App) *ExpirationOverlay {
	e := &ExpirationOverlay{
		app:   app,
		Flex:  tview.NewFlex(),
		List:  tview.NewList(),
		Text:  tview.NewTextView(),
		Index: 4,
	}

	e.Flex.SetBackgroundColor(app.Config.Style.Background)
	e.List.SetMainTextColor(app.Config.Style.Text)
	e.List.SetBackgroundColor(app.Config.Style.Background)
	e.List.SetSelectedTextColor(app.Config.Style.ListSelectedText)
	e.List.SetSelectedBackgroundColor(app.Config.Style.ListSelectedBackground)
	e.List.ShowSecondaryText(false)
	e.List.SetHighlightFullLine(true)
	e.Text.SetBackgroundColor(app.Config.Style.Background)
	e.Text.SetTextColor(app.Config.Style.Text)
	e.Flex.SetDrawFunc(app.Config.ClearContent)

	e.Text.SetText("Select the expiration time you want pressing Enter.")
	items := []string{
		"5 minutes",
		"30 minutes",
		"1 hour",
		"6 hours",
		"1 day",
		"3 days",
		"7 days",
	}
	for _, item := range items {
		e.List.AddItem(item, "", 0, nil)
	}
	e.List.SetCurrentItem(4)
	return e
}

type ExpirationOverlay struct {
	app   *App
	Flex  *tview.Flex
	List  *tview.List
	Text  *tview.TextView
	Index int
}

func (e *ExpirationOverlay) Reset() {
	e.List.SetCurrentItem(4)
	e.Index = 4
}

func (e *ExpirationOverlay) Prev() {
	index := e.List.GetCurrentItem()
	if index-1 >= 0 {
		e.List.SetCurrentItem(index - 1)
	}
}

func (e *ExpirationOverlay) Next() {
	index := e.List.GetCurrentItem()
	if index+1 < e.List.GetItemCount() {
		e.List.SetCurrentItem(index + 1)
	}
}

func (e *ExpirationOverlay) Done() {
	index := e.List.GetCurrentItem()
	e.Index = index
	e.app.UI.SetFocus(PollFocus)
}

func (e *ExpirationOverlay) GetExpiration() int {
	exp := []int{
		60 * 5,
		60 * 30,
		60 * 60,
		60 * 60 * 6,
		60 * 60 * 24,
		60 * 60 * 24 * 3,
		60 * 60 * 24 * 7,
	}
	return exp[e.Index]
}

func (e *ExpirationOverlay) InputHandler(event *tcell.EventKey) {
	if event.Key() == tcell.KeyRune {
		switch event.Rune() {
		case 'j', 'J':
			e.Next()
		case 'k', 'K':
			e.Prev()
		case 'q', 'Q':
			e.app.UI.SetFocus(PollFocus)
		}
	} else {
		switch event.Key() {
		case tcell.KeyEnter:
			e.Done()
		case tcell.KeyUp:
			e.Prev()
		case tcell.KeyDown:
			e.Next()
		case tcell.KeyESC:
			e.app.UI.SetFocus(PollFocus)
		}
	}
}
