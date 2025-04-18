package ping

import "time"

// Config holds the configuration for the ping server
type Config struct {
	TargetHost   string
	ListenAddr   string
	PingInterval time.Duration
	Verbose      bool
} 