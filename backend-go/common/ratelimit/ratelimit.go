package ratelimit

type Rule struct {
	Key      string
	PerMin   int
	Strategy string
}
