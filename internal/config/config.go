package config

type Config struct {
	TiDB      TiDBConfig
	Jina      JinaConfig
	Server    ServerConfig
	MCPServer MCPConfig
}

type TiDBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

type JinaConfig struct {
	APIURL string
	APIKey string
}

type ServerConfig struct {
	Port int
}

type MCPConfig struct {
	Port int
}

func Load() *Config {
	return &Config{
		TiDB: TiDBConfig{
			Host:     "192.168.1.13",
			Port:     4000,
			User:     "root",
			Password: "",
			Database: "vector_demo",
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
}