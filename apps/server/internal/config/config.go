package config

import (
	"log"
	"os"
)

// Config 汇总服务配置（M1 起扩展 DB/Meili/Casdoor 等）。
type Config struct {
	Env  string
	Port string
}

// Load 从环境变量加载配置（生产可用 env；本地默认值对齐 docker-compose）。
func Load() *Config {
	return &Config{
		Env:  getEnv("YOURTJ_ENV", "development"),
		Port: getEnv("YOURTJ_PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if fallback != "" {
		log.Printf("config: %s not set, using default %q", key, fallback)
	}
	return fallback
}
