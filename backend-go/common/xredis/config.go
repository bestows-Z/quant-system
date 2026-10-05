package xredis

type Config struct {
	Host string
	Type string
	Pass string
}

func Key(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, part := range parts[1:] {
		out += ":" + part
	}
	return out
}
