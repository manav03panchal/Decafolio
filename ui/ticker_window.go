package ui

import (
	g "github.com/AllenDang/giu"
)

type TickerWindow struct {
	symbol  string
	isOpen  bool
	posX    float32
	posY    float32
	onClose func(string)
}

func NewTickerWindow(symbol string, onClose func(string)) *TickerWindow {
	static := 0
	static++

	return &TickerWindow{
		symbol:  symbol,
		isOpen:  true,
		posX:    float32(50 + (static * 30)),
		posY:    float32(50 + (static * 30)),
		onClose: onClose,
	}
}

func (tw *TickerWindow) Render() {
	if !tw.isOpen {
		tw.onClose(tw.symbol)
		return
	}

	g.Window(tw.symbol).IsOpen(&tw.isOpen).Pos(tw.posX, tw.posY).Size(300, 200).Layout(
		g.Label(tw.symbol),
		g.Separator(),
		g.Button("Close").OnClick(func() {
			tw.isOpen = false
		}),
	)
}
