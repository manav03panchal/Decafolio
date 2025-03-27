// data/models.go
package data

import (
	"sync"
	"time"
)

// QuoteData represents the current market data for display purposes
type QuoteData struct {
	Symbol    string
	BidPrice  float64
	AskPrice  float64
	Timestamp time.Time
}

// QuoteCache is a simple in-memory cache for the latest quote
// No historical data is stored, just the current state
type QuoteCache struct {
	data  map[string]QuoteData
	mutex sync.RWMutex
}

// NewQuoteCache creates a new QuoteCache
func NewQuoteCache() *QuoteCache {
	return &QuoteCache{
		data: make(map[string]QuoteData),
	}
}

// Update sets the latest quote for a symbol
func (c *QuoteCache) Update(symbol string, bid, ask float64, timestamp time.Time) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[symbol] = QuoteData{
		Symbol:    symbol,
		BidPrice:  bid,
		AskPrice:  ask,
		Timestamp: timestamp,
	}
}

// Get retrieves the current quote for a symbol
func (c *QuoteCache) Get(symbol string) (QuoteData, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	data, exists := c.data[symbol]
	return data, exists
}
