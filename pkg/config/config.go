package config

import (
	"time"
)

type Config struct {
	APIBindPort int
	DataDir     string
	AdminToken  string

	ClusterNatsUrl           string
	ClusterNatsPrefix        string
	ClusterHeartbeatInterval time.Duration

	LoadApps      bool
	UseSystemDeno bool // Use the system installed deno rather than the version downloaded automatically

	FunctionRunTimeout         time.Duration
	HTTPGatewayResponseTimeout time.Duration
	SanboxJobInitTimeout       time.Duration
	SandboxCleanupInterval     time.Duration
	SandboxFunctionKeepAlive   time.Duration
	SandboxJobStartTimeout     time.Duration
	SandboxJobStopTimeout      time.Duration
}

func NewConfig() *Config {
	return &Config{
		LoadApps:                   true,
		ClusterNatsUrl:             "nats://localhost:4222",
		ClusterNatsPrefix:          "mls",
		ClusterHeartbeatInterval:   2 * time.Second,
		FunctionRunTimeout:         1 * time.Minute,
		HTTPGatewayResponseTimeout: 10 * time.Second,
		SanboxJobInitTimeout:       10 * time.Second,
		SandboxJobStartTimeout:     10 * time.Second,
		SandboxJobStopTimeout:      2 * time.Second,
		SandboxCleanupInterval:     1 * time.Minute,
		SandboxFunctionKeepAlive:   2 * time.Minute,
	}
}
