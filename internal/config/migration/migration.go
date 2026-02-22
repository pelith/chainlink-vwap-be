package migration

type Config struct {
	PostgreSQL PostgreSQL `mapstructure:"postgresql" structs:"postgresql"`
}

type PostgreSQL struct {
	Database string `mapstructure:"database" structs:"database"`
	Host     string `mapstructure:"host" structs:"host"`
	Port     string `mapstructure:"port" structs:"port"`
	User     string `mapstructure:"user" structs:"user"`
	Password string `mapstructure:"password" structs:"password"`
}
