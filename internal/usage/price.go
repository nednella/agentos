package usage

import (
	"math"
	"strings"
)

// PricesAsOf is the day the prices below were taken from Anthropic's API list prices.
const PricesAsOf = "2026-10-06"

// price is a model's list price in US dollars per million tokens. A cache write costs 1.25 times the
// input price when it is kept 5 minutes and twice it when kept an hour.
type price struct {
	input, output, cacheRead float64
}

// prices is matched by prefix, first match wins, so a model comes before the shorter name it starts with.
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
}

// Claude Haiku 5.5 costs five times more for a prompt over this many tokens.
const haikuLongPrompt = 100_000

func priceOf(model string, prompt int64) (price, bool) {
	for _, p := range prices {
		if !strings.HasPrefix(model, p.prefix) {
			continue
		}
		if p.prefix == "claude-haiku-5-5" && prompt > haikuLongPrompt {
			return price{p.input * 5, p.output * 5, p.cacheRead * 5}, true
		}
		return p.price, true
	}
	return price{}, false
}

// cost is what one API response cost in millionths of a US dollar, 0 for a model with no known price.
// Fast mode costs twice the standard price.
func cost(model string, u apiUsage) int64 {
	write5m, write1h := u.CacheCreation.Ephemeral5m, u.CacheCreation.Ephemeral1h
	if write5m+write1h == 0 {
		write5m = u.CacheCreationInputTokens
	}
	p, ok := priceOf(model, u.InputTokens+u.CacheReadInputTokens+u.CacheCreationInputTokens)
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
