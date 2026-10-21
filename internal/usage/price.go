package usage

import (
	"math"
	"strings"
	"time"
)

// PricesAsOf is the day the prices below were taken from Anthropic's and Google's API list prices.
const PricesAsOf = "2026-10-20"

// price is a model's list price in US dollars per million tokens. A cache write costs 1.25 times the
// input price when it is kept 5 minutes and twice it when kept an hour.
type price struct {
	input, output, cacheRead float64
}

// prices is matched by prefix, first match wins, so a model comes before the shorter name it starts with. A model
// id from agy ends in its effort (gemini-4-argon-medium), which the prefix match passes over.
var prices = []struct {
	prefix string
	price
}{
	{"claude-fable-5-1", price{10, 50, 0.25}},
	{"claude-mythos-5-1", price{10, 50, 0.25}},
	{"claude-fable-5", price{10, 50, 1}},
	{"claude-mythos-5", price{10, 50, 1}},
	{"claude-opus-5-5", price{4, 20, 0.2}},
	{"claude-opus-5", price{5, 25, 0.5}},
	{"claude-opus-4-8", price{5, 25, 0.5}},
	{"claude-opus-4-7", price{5, 25, 0.5}},
	{"claude-opus-4-6", price{5, 25, 0.5}},
	{"claude-opus-4-5", price{5, 25, 0.5}},
	{"claude-opus-4", price{15, 75, 1.5}},
	{"claude-sonnet-5", price{2, 10, 0.2}},
	{"claude-sonnet-4", price{3, 15, 0.3}},
	{"claude-haiku-5-5", price{0.1, 0.5, 0.01}},
	{"claude-haiku-4-5", price{1, 5, 0.1}},
	{"gemini-4-argon", price{2, 10, 0.1}},
	{"gemini-3.8-flash", price{0.75, 3.75, 0}}, // Google's list gives no cached price for it
}

// argonRise is the first day Gemini 4 Argon costs twice its launch price.
var argonRise = time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)

// Claude Haiku 5.5 costs five times more for a prompt over this many tokens.
const haikuLongPrompt = 100_000

// priceOf is the price of model for a response at the time given, with a prompt of that many tokens.
func priceOf(model string, prompt int64, at time.Time) (price, bool) {
	for _, p := range prices {
		if !strings.HasPrefix(model, p.prefix) {
			continue
		}
		if p.prefix == "claude-haiku-5-5" && prompt > haikuLongPrompt {
			return price{p.input * 5, p.output * 5, p.cacheRead * 5}, true
		}
		if p.prefix == "gemini-4-argon" && !at.Before(argonRise) {
			return price{p.input * 2, p.output * 2, p.cacheRead * 2}, true
		}
		return p.price, true
	}
	return price{}, false
}

// cost is what one API response at the time given cost in millionths of a US dollar, 0 for a model with no known
// price. Fast mode costs twice the standard price.
func cost(model string, u apiUsage, at time.Time) int64 {
	write5m, write1h := u.CacheCreation.Ephemeral5m, u.CacheCreation.Ephemeral1h
	if write5m+write1h == 0 {
		write5m = u.CacheCreationInputTokens
	}
	p, ok := priceOf(model, u.InputTokens+u.CacheReadInputTokens+u.CacheCreationInputTokens, at)
	if !ok {
		return 0
	}
	// A price in dollars per million tokens is also millionths of a dollar per token.
	c := float64(u.InputTokens)*p.input + float64(u.OutputTokens)*p.output + float64(u.CacheReadInputTokens)*p.cacheRead +
		float64(write5m)*p.input*1.25 + float64(write1h)*p.input*2
	if u.Speed == "fast" {
		c *= 2
	}
	return int64(math.Round(c))
}
