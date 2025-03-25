package ui

import (
	g "github.com/AllenDang/giu"
)

type App struct {
	masterWindow        *g.MasterWindow
	tickerWindows       map[string]*TickerWindow
	showAddTickerDialog bool
	newTickerSymbol     string
}

func NewApp() *App {
	return &App{
		tickerWindows: make(map[string]*TickerWindow),
	}
}

func (a *App) Run() {
	a.masterWindow = g.NewMasterWindow("Decafolio", 800, 600, 0)
	a.masterWindow.Run(a.loop)
}

func (a *App) loop() {
	g.MainMenuBar().Layout(
		g.Menu("File").Layout(
			g.MenuItem("Add Ticker").OnClick(func() {
				a.showAddTickerDialog = true
			}),
			g.Separator(),
		),
	).Build()

	// g.Window("My Tickers").Pos(10, 30).Size(300, 400).Layout()

	if a.showAddTickerDialog {
		g.Window("Add Ticker").IsOpen(&a.showAddTickerDialog).Flags(g.WindowFlagsNoResize).Size(250, 100).Layout(
			g.Label("Enter Ticker Symbol:"),
			g.InputText(&a.newTickerSymbol),
			g.Row(
				g.Button("Add").OnClick(func() {
					if a.newTickerSymbol != "" {
						a.AddTickerWindow(a.newTickerSymbol)
						a.newTickerSymbol = ""
						a.showAddTickerDialog = false
					}
				}),
				g.Button("Cancel").OnClick(func() {
					a.showAddTickerDialog = false
				}),
			),
		)
	}

	for _, window := range a.tickerWindows {
		window.Render()
	}
}

func (a *App) AddTickerWindow(symbol string) {
	if _, exists := a.tickerWindows[symbol]; exists {
		return
	}

	a.tickerWindows[symbol] = NewTickerWindow(symbol, func(s string) {
		delete(a.tickerWindows, s)
	})
}
