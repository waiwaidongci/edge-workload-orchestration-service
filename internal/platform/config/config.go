package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTP      HTTPConfig
	Scheduler SchedulerConfig
	Storage   StorageConfig
}

type HTTPConfig struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	MaxBodyBytes    int64
	RatePerSecond   int
}

type SchedulerConfig struct {
	Interval         time.Duration
	LeaseDuration    time.Duration
	HeartbeatTimeout time.Duration
	MaxAttempts      int
	BaseBackoff      time.Duration
}

type StorageConfig struct {
	PostgresDSN string
	RedisAddr   string
}

func Default() Config {
	return Config{
		HTTP:      HTTPConfig{Address: ":8090", ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, ShutdownTimeout: 10 * time.Second, MaxBodyBytes: 1 << 20, RatePerSecond: 100},
		Scheduler: SchedulerConfig{Interval: 250 * time.Millisecond, LeaseDuration: 30 * time.Second, HeartbeatTimeout: 45 * time.Second, MaxAttempts: 4, BaseBackoff: time.Second},
		Storage:   StorageConfig{PostgresDSN: "postgres://edge:edge@localhost:5432/edge?sslmode=disable", RedisAddr: "localhost:6379"},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path != "" {
		if err := applyYAML(&cfg, path); err != nil {
			return Config{}, err
		}
	}
	applyEnvironment(&cfg)
	return cfg, validate(cfg)
}

func applyYAML(cfg *Config, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config: %w", err)
	}
	defer file.Close()
	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid config line %q", line)
		}
		if err := set(cfg, section+"."+strings.TrimSpace(parts[0]), strings.Trim(strings.TrimSpace(parts[1]), "\"'")); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan config: %w", err)
	}
	return nil
}

func applyEnvironment(cfg *Config) {
	values := map[string]string{
		"http.address": os.Getenv("EDGE_HTTP_ADDRESS"), "http.read_timeout": os.Getenv("EDGE_HTTP_READ_TIMEOUT"),
		"http.write_timeout": os.Getenv("EDGE_HTTP_WRITE_TIMEOUT"), "http.shutdown_timeout": os.Getenv("EDGE_HTTP_SHUTDOWN_TIMEOUT"),
		"http.max_body_bytes": os.Getenv("EDGE_HTTP_MAX_BODY_BYTES"), "http.rate_per_second": os.Getenv("EDGE_HTTP_RATE_PER_SECOND"),
		"scheduler.interval": os.Getenv("EDGE_SCHEDULER_INTERVAL"), "scheduler.lease_duration": os.Getenv("EDGE_LEASE_DURATION"),
		"scheduler.heartbeat_timeout": os.Getenv("EDGE_HEARTBEAT_TIMEOUT"), "scheduler.max_attempts": os.Getenv("EDGE_MAX_ATTEMPTS"),
		"scheduler.base_backoff": os.Getenv("EDGE_BASE_BACKOFF"), "storage.postgres_dsn": os.Getenv("EDGE_POSTGRES_DSN"),
		"storage.redis_addr": os.Getenv("EDGE_REDIS_ADDR"),
	}
	for key, value := range values {
		if value != "" {
			_ = set(cfg, key, value)
		}
	}
}

func set(cfg *Config, key, value string) error {
	duration := func(target *time.Duration) error {
		parsed, err := time.ParseDuration(value)
		if err == nil {
			*target = parsed
		}
		return err
	}
	integer := func(target *int) error {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			*target = parsed
		}
		return err
	}
	switch key {
	case "http.address":
		cfg.HTTP.Address = value
	case "http.read_timeout":
		return duration(&cfg.HTTP.ReadTimeout)
	case "http.write_timeout":
		return duration(&cfg.HTTP.WriteTimeout)
	case "http.shutdown_timeout":
		return duration(&cfg.HTTP.ShutdownTimeout)
	case "http.max_body_bytes":
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		cfg.HTTP.MaxBodyBytes = parsed
	case "http.rate_per_second":
		return integer(&cfg.HTTP.RatePerSecond)
	case "scheduler.interval":
		return duration(&cfg.Scheduler.Interval)
	case "scheduler.lease_duration":
		return duration(&cfg.Scheduler.LeaseDuration)
	case "scheduler.heartbeat_timeout":
		return duration(&cfg.Scheduler.HeartbeatTimeout)
	case "scheduler.max_attempts":
		return integer(&cfg.Scheduler.MaxAttempts)
	case "scheduler.base_backoff":
		return duration(&cfg.Scheduler.BaseBackoff)
	case "storage.postgres_dsn":
		cfg.Storage.PostgresDSN = value
	case "storage.redis_addr":
		cfg.Storage.RedisAddr = value
	default:
		return fmt.Errorf("unknown configuration key %q", key)
	}
	return nil
}

func validate(cfg Config) error {
	if cfg.HTTP.Address == "" || cfg.HTTP.MaxBodyBytes <= 0 || cfg.HTTP.RatePerSecond <= 0 {
		return fmt.Errorf("invalid HTTP configuration")
	}
	if cfg.Scheduler.Interval <= 0 || cfg.Scheduler.LeaseDuration <= 0 || cfg.Scheduler.MaxAttempts < 1 {
		return fmt.Errorf("invalid scheduler configuration")
	}
	return nil
}
