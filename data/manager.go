// data/manager.go
package data

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata/stream"
)

// QuoteCallback is a function that will be called when a quote is received
type QuoteCallback func(symbol string, quote stream.CryptoQuote)

// StreamManager handles the WebSocket connection to Alpaca
type StreamManager struct {
	client             *stream.CryptoClient
	ctx                context.Context
	cancel             context.CancelFunc
	subscribers        map[string][]QuoteCallback // Map of symbol -> callbacks
	subscribersLock    sync.RWMutex
	connected          bool
	connectLock        sync.Mutex
	quoteHandlerMutex  sync.RWMutex
	globalQuoteHandler func(quote stream.CryptoQuote)
}

// NewStreamManager creates a new StreamManager
func NewStreamManager() *StreamManager {
	ctx, cancel := context.WithCancel(context.Background())
	sm := &StreamManager{
		ctx:         ctx,
		cancel:      cancel,
		subscribers: make(map[string][]QuoteCallback),
		connected:   false,
	}

	// Create the global quote handler function
	sm.globalQuoteHandler = func(quote stream.CryptoQuote) {
		sm.handleQuote(quote)
	}

	return sm
}

// Run starts the StreamManager with reconnection capability
func (sm *StreamManager) Run() {
	go func() {
		for {
			if err := sm.Connect(); err != nil {
				fmt.Printf("[StreamManager] Connection error: %v, retrying in 3 seconds...\n", err)
				time.Sleep(3 * time.Second)
				continue
			}

			// Wait for termination
			select {
			case <-sm.ctx.Done():
				fmt.Println("[StreamManager] Context canceled, shutting down")
				return
			case err := <-sm.client.Terminated():
				if err != nil {
					fmt.Printf("[StreamManager] Connection terminated with error: %v, reconnecting...\n", err)
				} else {
					fmt.Println("[StreamManager] Connection terminated normally, reconnecting...")
				}
				sm.connected = false
				time.Sleep(3 * time.Second)
			}
		}
	}()
}

// Connect establishes the WebSocket connection if not already connected
func (sm *StreamManager) Connect() error {
	sm.connectLock.Lock()
	defer sm.connectLock.Unlock()

	fmt.Println("[StreamManager] Connect called")

	if sm.connected {
		fmt.Println("[StreamManager] Already connected, skipping connection setup")
		return nil
	}

	fmt.Println("[StreamManager] Creating new CryptoClient")
	// Create a client with default handlers, using SIP for faster data
	// You can also try marketdata.FTXUS for FTX data which is used in the Python examples
	sm.client = stream.NewCryptoClient(marketdata.US,
		stream.WithLogger(stream.DefaultLogger()),
		// Pass our global handler function
		stream.WithCryptoQuotes(sm.globalQuoteHandler),
	)

	fmt.Println("[StreamManager] Connecting to Alpaca...")
	if err := sm.client.Connect(sm.ctx); err != nil {
		fmt.Printf("[StreamManager] Connection failed: %v\n", err)
		return fmt.Errorf("failed to connect to Alpaca: %w", err)
	}
	fmt.Println("[StreamManager] Connected successfully!")

	// Start a goroutine to handle connection termination
	go func() {
		fmt.Println("[StreamManager] Starting termination monitor")
		err := <-sm.client.Terminated()
		if err != nil {
			fmt.Printf("[StreamManager] Alpaca connection terminated with error: %v\n", err)
		} else {
			fmt.Println("[StreamManager] Alpaca connection terminated normally")
		}
		sm.connected = false
	}()

	sm.connected = true
	fmt.Println("[StreamManager] Connection setup complete")
	return nil
}

// Subscribe adds a callback for a specific symbol
func (sm *StreamManager) Subscribe(symbol string, callback QuoteCallback) error {
	fmt.Printf("[StreamManager] Subscribing to %s\n", symbol)

	// Ensure we're connected
	if err := sm.Connect(); err != nil {
		fmt.Printf("[StreamManager] Connection error: %v\n", err)
		return err
	}

	// Add the callback to our subscribers
	sm.subscribersLock.Lock()
	defer sm.subscribersLock.Unlock()

	if _, exists := sm.subscribers[symbol]; !exists {
		fmt.Printf("[StreamManager] First subscription for %s, creating handler list\n", symbol)
		sm.subscribers[symbol] = make([]QuoteCallback, 0)

		// If this is a new symbol, ensure we're subscribed to it
		fmt.Printf("[StreamManager] Calling SubscribeToQuotes for %s\n", symbol)
		if err := sm.client.SubscribeToQuotes(sm.globalQuoteHandler, symbol); err != nil {
			fmt.Printf("[StreamManager] Failed to subscribe to %s: %v\n", symbol, err)
			return fmt.Errorf("failed to subscribe to %s: %w", symbol, err)
		}
		fmt.Printf("[StreamManager] Successfully subscribed to %s\n", symbol)
	} else {
		fmt.Printf("[StreamManager] Already subscribed to %s, adding new callback\n", symbol)
	}

	sm.subscribers[symbol] = append(sm.subscribers[symbol], callback)
	fmt.Printf("[StreamManager] Now have %d callbacks for %s\n", len(sm.subscribers[symbol]), symbol)
	return nil
}

// Unsubscribe removes a callback for a specific symbol
func (sm *StreamManager) Unsubscribe(symbol string, callback QuoteCallback) {
	sm.subscribersLock.Lock()
	defer sm.subscribersLock.Unlock()

	fmt.Printf("[StreamManager] Unsubscribing from %s\n", symbol)

	callbacks, exists := sm.subscribers[symbol]
	if !exists {
		fmt.Printf("[StreamManager] No subscribers for %s, nothing to unsubscribe\n", symbol)
		return
	}

	// Find and remove the specific callback
	callbackFound := false
	for i, cb := range callbacks {
		if fmt.Sprintf("%p", cb) == fmt.Sprintf("%p", callback) {
			// Remove this callback
			fmt.Printf("[StreamManager] Removing callback for %s\n", symbol)
			sm.subscribers[symbol] = append(callbacks[:i], callbacks[i+1:]...)
			callbackFound = true
			break
		}
	}

	if !callbackFound {
		fmt.Printf("[StreamManager] Callback not found for %s\n", symbol)
	}

	// If no more subscribers for this symbol, unsubscribe from Alpaca
	if len(sm.subscribers[symbol]) == 0 {
		fmt.Printf("[StreamManager] No more subscribers for %s, unsubscribing from Alpaca\n", symbol)
		delete(sm.subscribers, symbol)

		// Properly unsubscribe from the Alpaca WebSocket
		err := sm.client.UnsubscribeFromQuotes(symbol)
		if err != nil {
			fmt.Printf("[StreamManager] Error unsubscribing from %s: %v\n", symbol, err)
		} else {
			fmt.Printf("[StreamManager] Successfully unsubscribed from %s\n", symbol)
		}
	} else {
		fmt.Printf("[StreamManager] Still have %d subscribers for %s\n", len(sm.subscribers[symbol]), symbol)
	}
}

// handleQuote distributes incoming quotes to the appropriate callbacks
func (sm *StreamManager) handleQuote(quote stream.CryptoQuote) {
	symbol := quote.Symbol

	// Log incoming data
	fmt.Printf("[StreamManager] Received quote for %s: Bid=%.2f, Ask=%.2f\n",
		symbol, quote.BidPrice, quote.AskPrice)

	sm.subscribersLock.RLock()
	callbacks, exists := sm.subscribers[symbol]
	numCallbacks := len(callbacks)
	sm.subscribersLock.RUnlock()

	if !exists {
		fmt.Printf("[StreamManager] No subscribers for %s, dropping data\n", symbol)
		return // No one is listening for this symbol
	}

	fmt.Printf("[StreamManager] Notifying %d subscribers for %s\n", numCallbacks, symbol)

	// Call all callbacks for this symbol
	for i, callback := range callbacks {
		fmt.Printf("[StreamManager] Calling callback #%d for %s\n", i+1, symbol)
		callback(symbol, quote)
	}
}

// Close terminates the WebSocket connection
func (sm *StreamManager) Close() {
	// First unsubscribe from all symbols
	sm.subscribersLock.Lock()
	symbols := make([]string, 0, len(sm.subscribers))
	for symbol := range sm.subscribers {
		symbols = append(symbols, symbol)
	}
	sm.subscribersLock.Unlock()

	// Unsubscribe from each symbol
	for _, symbol := range symbols {
		fmt.Printf("[StreamManager] Closing: Unsubscribing from %s\n", symbol)
		err := sm.client.UnsubscribeFromQuotes(symbol)
		if err != nil {
			fmt.Printf("[StreamManager] Error unsubscribing from %s during shutdown: %v\n", symbol, err)
		}
	}

	// Then cancel context and mark as disconnected
	sm.cancel()
	sm.connected = false
	fmt.Println("[StreamManager] WebSocket connection closed")
}
