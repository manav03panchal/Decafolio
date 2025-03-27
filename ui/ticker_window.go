// ui/ticker_window.go
package ui

import (
	"fmt"

	g "github.com/AllenDang/giu"
	"github.com/manav03panchal/decafolio/data"
)

type TickerWindow struct {
	symbol     string
	isOpen     bool
	posX       float32
	posY       float32
	onClose    func(string)
	lastUpdate string
	updateFunc func() // Store the update callback for cleanup
}

func NewTickerWindow(symbol string, onClose func(string)) *TickerWindow {
	static := 0
	static++

	tw := &TickerWindow{
		symbol:     symbol,
		isOpen:     true,
		posX:       float32(50 + (static * 30)),
		posY:       float32(50 + (static * 30)),
		onClose:    onClose,
		lastUpdate: "Connecting...",
	}

	// Create update callback
	tw.updateFunc = func() {
		fmt.Printf("[TickerWindow] Update callback triggered for %s\n", tw.symbol)
		// Notify GUI to refresh
		g.Update()
	}

	// Subscribe to real-time quotes
	client := data.GetAlpacaClient()
	fmt.Printf("[TickerWindow] Subscribing to quotes for %s\n", tw.symbol)
	err := client.SubscribeToQuotes(tw.symbol, tw.updateFunc)

	if err != nil {
		fmt.Printf("[TickerWindow] Error subscribing to %s: %v\n", tw.symbol, err)
		tw.lastUpdate = fmt.Sprintf("Error: %v", err)
	} else {
		fmt.Printf("[TickerWindow] Successfully subscribed to %s\n", tw.symbol)
	}

	return tw
}

func (tw *TickerWindow) Render() {
	if !tw.isOpen {
		// Clean up before closing
		fmt.Printf("[TickerWindow] Closing window for %s\n", tw.symbol)
		client := data.GetAlpacaClient()
		client.UnsubscribeFromQuotes(tw.symbol, tw.updateFunc)
		fmt.Printf("[TickerWindow] Unsubscribed from quotes for %s\n", tw.symbol)

		tw.onClose(tw.symbol)
		return
	}

	// Get current quote data
	client := data.GetAlpacaClient()
	quoteData, exists := client.GetQuote(tw.symbol)

	var displaySymbol string
	var bid, ask float64
	var timestampStr string

	if exists {
		displaySymbol = quoteData.Symbol
		bid = quoteData.BidPrice
		ask = quoteData.AskPrice
		timestampStr = quoteData.Timestamp.String()

		// Only log when timestamp changes (i.e., new data)
		if timestampStr != tw.lastUpdate {
			fmt.Printf("[TickerWindow] New data for %s: bid=%.2f, ask=%.2f, time=%s\n",
				tw.symbol, bid, ask, timestampStr)
			tw.lastUpdate = timestampStr
		}
	} else {
		displaySymbol = tw.symbol
		timestampStr = "Waiting for data..."

		// No logging here at all to prevent spam
	}

	g.Window(tw.symbol).IsOpen(&tw.isOpen).Pos(tw.posX, tw.posY).Size(300, 200).Layout(
		g.Label(fmt.Sprintf("Symbol: %s", displaySymbol)),
		g.Separator(),
		g.Label(fmt.Sprintf("Mid Price: $%.2f", (bid+ask)/2)),
		g.Label(fmt.Sprintf("Bid: $%.2f", bid)),
		g.Label(fmt.Sprintf("Ask: $%.2f", ask)),
		g.Label(fmt.Sprintf("Spread: $%.2f", ask-bid)),
		g.Label(fmt.Sprintf("Last Update: %s", timestampStr)),
		g.Row(
			g.Button("Close").OnClick(func() {
				tw.isOpen = false
			}),
		),
	)
}
