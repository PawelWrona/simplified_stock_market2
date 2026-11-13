package config

type Config struct {
	Host string
	Port string
}

func New() *Config {
	return &Config{
		Host: "0.0.0.0",
		Port: "8080",
	}
}
