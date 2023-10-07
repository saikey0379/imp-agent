package config

// Loader 定义统一的配置加载接口
type Loader interface {
	Load() (*Config, error)
	Save(*Config) error
}

// Config config 数据结构体
type Config struct {
	Agent struct {
		Server       string `ini:"server"`
		Listen       string `ini:"listen"`
		Developer    int    `ini:"developer"`
		Interval     int    `ini:"interval"`
		PreScript    string `ini:"preScript"`
		ReportPolicy string `ini:"reportPolicy"`
	}
	Logger struct {
		Color   bool   `ini:"color"`
		Level   string `ini:"level"`
		LogFile string `ini:"logFile"`
		Logger  Logger
	}
}

type Logger struct {
	Color   bool   `ini:"color"`
	Level   string `ini:"level"`
	LogFile string `ini:"logFile"`
}
