// data/alpaca.go
package data

import (
	"fmt"
	"sync"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata/stream"
)

// AlpacaClient manages the connection to Alpaca
type AlpacaClient struct {
	streamManager *StreamManager
	quoteCache    *QuoteCache
	callbacks     map[string][]func()
	callbackLock  sync.RWMutex
}

// Global client instance
var (
	client     *AlpacaClient
	clientOnce sync.Once
)

// GetAlpacaClient returns the singleton AlpacaClient instance
func GetAlpacaClient() *AlpacaClient {
	clientOnce.Do(func() {
		client = &AlpacaClient{
			streamManager: NewStreamManager(),
			quoteCache:    NewQuoteCache(),
			callbacks:     make(map[string][]func()),
		}
	})
	return client
}

// GetStreamManager returns the stream manager
func (ac *AlpacaClient) GetStreamManager() *StreamManager {
	return ac.streamManager
}

// SubscribeToQuotes subscribes to quote updates for a symbol
func (ac *AlpacaClient) SubscribeToQuotes(symbol string, onUpdate func()) error {
	// Register the UI callback
	ac.registerCallback(symbol, onUpdate)

	// Create a callback function that updates the cache and notifies the UI
	streamCallback := func(symbol string, cryptoQuote stream.CryptoQuote) {
		// Extract price information
		bid := cryptoQuote.BidPrice
		ask := cryptoQuote.AskPrice
		timestamp := time.Time(cryptoQuote.Timestamp)

		// Update the cache
		ac.quoteCache.Update(symbol, bid, ask, timestamp)

		// Notify all registered UI components
		ac.notifyCallbacks(symbol)
	}

	// Subscribe to the stream
	err := ac.streamManager.Subscribe(symbol, streamCallback)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", symbol, err)
	}

	return nil
}

// registerCallback adds a UI update callback for a symbol
func (ac *AlpacaClient) registerCallback(symbol string, callback func()) {
	if callback == nil {
		return
	}

	ac.callbackLock.Lock()
	defer ac.callbackLock.Unlock()

	if _, exists := ac.callbacks[symbol]; !exists {
		ac.callbacks[symbol] = make([]func(), 0)
	}

	ac.callbacks[symbol] = append(ac.callbacks[symbol], callback)
}

// notifyCallbacks calls all registered UI callbacks for a symbol
func (ac *AlpacaClient) notifyCallbacks(symbol string) {
	ac.callbackLock.RLock()
	callbacks := ac.callbacks[symbol]
	numCallbacks := len(callbacks)
	ac.callbackLock.RUnlock()

	fmt.Printf("[AlpacaClient] Notifying %d UI callbacks for %s\n", numCallbacks, symbol)

	for i, callback := range callbacks {
		fmt.Printf("[AlpacaClient] Calling UI callback #%d for %s\n", i+1, symbol)
		callback()
	}
}

// UnsubscribeFromQuotes removes a UI callback and unsubscribes if no more listeners
func (ac *AlpacaClient) UnsubscribeFromQuotes(symbol string, callback func()) {
	ac.callbackLock.Lock()

	fmt.Printf("[AlpacaClient] Unsubscribing from %s\n", symbol)

	callbackFound := false
	if callbacks, exists := ac.callbacks[symbol]; exists {
		// Find and remove the specific callback
		for i, cb := range callbacks {
			if fmt.Sprintf("%p", cb) == fmt.Sprintf("%p", callback) {
				fmt.Printf("[AlpacaClient] Removing UI callback for %s\n", symbol)
				ac.callbacks[symbol] = append(callbacks[:i], callbacks[i+1:]...)
				callbackFound = true
				break
			}
		}

		// If no more callbacks for this symbol, we can clean up
		if len(ac.callbacks[symbol]) == 0 {
			fmt.Printf("[AlpacaClient] No more UI callbacks for %s, removing from registry\n", symbol)
			delete(ac.callbacks, symbol)
		} else {
			fmt.Printf("[AlpacaClient] Still have %d UI callbacks for %s\n", len(ac.callbacks[symbol]), symbol)
		}
	} else {
		fmt.Printf("[AlpacaClient] No registered UI callbacks for %s\n", symbol)
	}

	if !callbackFound {
		fmt.Printf("[AlpacaClient] UI callback not found for %s\n", symbol)
	}

	ac.callbackLock.Unlock()
}

// GetQuote returns the current quote data for a symbol
func (ac *AlpacaClient) GetQuote(symbol string) (QuoteData, bool) {
	return ac.quoteCache.Get(symbol)
}

// Close terminates all connections and cleans up resources
func (ac *AlpacaClient) Close() {
	fmt.Println("[AlpacaClient] Closing all connections and cleaning up resources")

	// First, clear all callbacks
	ac.callbackLock.Lock()
	ac.callbacks = make(map[string][]func())
	ac.callbackLock.Unlock()

	// Then close the stream manager
	if ac.streamManager != nil {
		ac.streamManager.Close()
	}

	fmt.Println("[AlpacaClient] All resources cleaned up")
}
