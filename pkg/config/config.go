//
// Created by Javid Rzayev on 12.08.26.
//

package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Host            string
	Port            int
	DriPath         string
	AccelPath       string
	SysfsDriPath    string
	SysfsAccelPath  string
	CgroupRootPath  string
	CriSocketPath   string
	Interval        time.Duration
	Debug           bool
	AppVersion      string
	ShutdownTimeout time.Duration
	CriTimeout      time.Duration
	CacheTTL        time.Duration
	QueueSize       int
}

func Load() Config {
	cfg := Config{
		Host:            getEnv("KUBENPU_HOST", "0.0.0.0"),
		Port:            getEnvInt("KUBENPU_PORT", 8080),
		DriPath:         getEnv("KUBENPU_DRI_PATH", "/dev/dri"),
		AccelPath:       getEnv("KUBENPU_ACCEL_PATH", "/dev/accel"),
		SysfsDriPath:    getEnv("KUBENPU_SYSFS_DRI_PATH", "/sys/class/drm"),
		SysfsAccelPath:  getEnv("KUBENPU_SYSFS_ACCEL_PATH", "/sys/class/accel"),
		CgroupRootPath:  getEnv("KUBENPU_CGROUP_ROOT_PATH", "/sys/fs/cgroup"),
		CriSocketPath:   getEnv("KUBENPU_CRI_SOCKET_PATH", "/run/k3s/containerd/containerd.sock"),
		Interval:        getEnvTimeDuration("KUBENPU_INTERVAL", 5*time.Second),
		Debug:           getEnvInt("KUBENPU_DEBUG", 0) != 0,
		AppVersion:      getEnv("KUBENPU_APP_VERSION", "0.0.1"),
		ShutdownTimeout: getEnvTimeDuration("KUBENPU_SHUTDOWN_TIMEOUT", 5*time.Second),
		CriTimeout:      getEnvTimeDuration("KUBENPU_CRI_TIMEOUT", 5*time.Second),
		CacheTTL:        getEnvTimeDuration("KUBENPU_CACHE_TTL", 60*time.Second),
		QueueSize:       getEnvInt("KUBENPU_QUEUE_SIZE", 4096),
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}

	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	if key == "KUBENPU_PORT" && n > 65535 {
		return fallback
	}

	return n
}

func getEnvTimeDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}

	seconds, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}

	return time.Duration(seconds) * time.Second
}
