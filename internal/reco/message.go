package reco

import (
	"fmt"
	"strings"
)

const telegramLimit = 4000 // Telegram caps a message at 4096 characters

// Messages renders one plain-text message per market. Every message starts and ends with the disclaimer, so
// no part of the ranking is ever sent without it.
func Messages(rep Report) []string {
	out := make([]string, 0, len(rep.Markets))
	for _, m := range rep.Markets {
		var b strings.Builder
		fmt.Fprintf(&b, "%s\n\nDaily ranking %s - %s (%s)\n", rep.Disclaimer, rep.Date, m.Market, m.Source)
		switch {
		case m.Error != "":
			fmt.Fprintf(&b, "\nUnavailable today: %s\n", m.Error)
		case len(m.Ranking.Top) == 0:
			b.WriteString("\nNo symbol passed the filters today.\n")
		default:
			b.WriteString("\n#  Symbol  Return  Daily vol  Score\n")
			for _, e := range m.Ranking.Top {
				fmt.Fprintf(
					&b,
					"%d. %s  %s%%  %s%%  %.2f\n",
					e.Rank,
					e.Symbol,
					e.ReturnPct,
					e.DailyVolatilityPct,
					e.Score,
				)
			}
			b.WriteString("\nScore = lookback return / daily volatility. It ranks past behaviour only.\n")
		}
		b.WriteString("\n" + rep.Disclaimer)
		text := b.String()
		if len(text) > telegramLimit {
			text = text[:telegramLimit]
		}
		out = append(out, text)
	}
	return out
}
