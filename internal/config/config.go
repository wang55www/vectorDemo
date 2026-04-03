package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	TiDB       TiDBConfig
	Embedding  EmbeddingConfig
	DashScope  DashScopeConfig
	Baidu      BaiduConfig
	Jina       JinaConfig
	Server     ServerConfig
	MCPServer  MCPConfig
}

type EmbeddingConfig struct {
	Provider string `toml:"provider"` // dashscope | baidu | jina
}

type DashScopeConfig struct {
	APIURL           string `toml:"api_url"`
	APIKey           string `toml:"api_key"`
	Model            string `toml:"model"`
	VectorDimension int    `toml:"vector_dimension"`
}

type BaiduConfig struct {
	APIKey    string `toml:"api_key"`
	SecretKey string `toml:"secret_key"`
	Model     string `toml:"model"`
}

type TiDBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

type JinaConfig struct {
	APIURL string `toml:"api_url"`
	APIKey string `toml:"api_key"`
	Proxy  string `toml:"proxy"`
}

type ServerConfig struct {
	Port int
}

type MCPConfig struct {
	Port int
}

// Load 从 TOML 配置文件加载配置
func Load() *Config {
	cfg := &Config{
		TiDB: TiDBConfig{
			Host:     "127.0.0.1",
			Port:     4000,
			User:     "root",
			Password: "",
			Database: "vector_demo",
		},
		Embedding: EmbeddingConfig{
			Provider: "dashscope", // 默认使用阿里云
		},
		DashScope: DashScopeConfig{
			APIURL:           "https://dashscope.aliyuncs.com/api/v1/services/embeddings/multimodal-embedding/multimodal-embedding",
			APIKey:           "",
			Model:            "qwen3-vl-embedding",
			VectorDimension: 2560,
		},
		Baidu: BaiduConfig{
			APIKey:    "",
			SecretKey: "",
			Model:     "embedding-v1",
		},
		Jina: JinaConfig{
			APIURL: "https://api.jina.ai/v1/embeddings",
			APIKey: "jina_1789bc1b3714423ba7a86e665ebdc0c1yLlcx4LOSC9kNzAD16Z8AvUpcozI",
		},
		Server: ServerConfig{
			Port: 8080,
		},
		MCPServer: MCPConfig{
			Port: 8081,
		},
	}

	// 尝试读取配置文件
	configFile := "config.toml"
	if envFile := os.Getenv("CONFIG_FILE"); envFile != "" {
		configFile = envFile
	}

	if _, err := os.Stat(configFile); err == nil {
		if _, err := toml.DecodeFile(configFile, cfg); err != nil {
			fmt.Printf("Warning: failed to load config file %s: %v\n", configFile, err)
			fmt.Println("Using default configuration")
		}
	}

	return cfg
}