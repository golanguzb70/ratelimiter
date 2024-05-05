package ratelimiter

type LeakyBucket struct {
	Method       string
	Path         string
	RequestLimit int
	DurationType string
	Type         string
	JWTKey       string
	AllowOnError bool
}

func (l *LeakyBucket) Validate() (string, bool) {
	switch l.Method {
	case "GET", "POST", "PUT", "DELETE":
	default:
		return "Method must be one of GET, POST, PUT, DELETE", false
	}

	switch {
	case l.RequestLimit < 1:
		return "RequestLimit must be greater than 0", false
	case l.DurationType != "second" && l.DurationType != "minute" && l.DurationType != "hour":
		return "DurationType must be one of second, minute, hour", false
	case l.Type != "ip" && l.Type != "per-user":
		return "Type must be one of ip, per-user", false
	}

	return "", true
}
