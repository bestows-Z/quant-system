package hub

import "fmt"

func SubscriptionKey(channel, symbol string, accountID uint64) string {
	if symbol != "" {
		return fmt.Sprintf("%s:symbol:%s", channel, symbol)
	}
	if accountID > 0 {
		return fmt.Sprintf("%s:account:%d", channel, accountID)
	}
	return channel
}
