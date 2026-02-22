package config

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type Config[T any] struct {
	Env   string
	Name  string `mapstructure:"name" structs:"name"`
	Pprof bool   `mapstructure:"pprof" structs:"pprof"`
	Log   struct {
		Level slog.Level `mapstructure:"level" structs:"level"`
	} `mapstructure:"log" structs:"log"`
	AppConfig T `mapstructure:"app_config" structs:"app_config"`
}

func Load[T any](env string) (*Config[T], error) {
	return LoadFromDir[T](env, "./config")
}

func LoadFromDir[T any](env string, configDir string) (*Config[T], error) {
	var cfg Config[T]

	v := viper.New()

	v.SetConfigFile(fmt.Sprintf("%s/base.yaml", configDir))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	v.SetConfigFile(fmt.Sprintf("%s/%s.yaml", configDir, env))

	if err := v.MergeInConfig(); err != nil {
		return nil, fmt.Errorf("merge config: %w", err)
	}

	// 用 YAML 讀入後的 key（dot notation，與 config 檔一致）對應環境變數，
	// 例如 app_config.http.addr → APP_CONFIG_HTTP_ADDR，這樣 docker-compose 的
	// environment 才會正確覆寫。Viper Unmarshal 只會用已寫入 viper 的值。
	for _, key := range v.AllKeys() {
		envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		if val := os.Getenv(envKey); val != "" {
			v.Set(key, val)
		}
	}

	hooks := mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		logLevelHookFunction,
		uuidHookFunction,
	)

	if err := v.Unmarshal(&cfg, viper.DecodeHook(hooks)); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.Env = env

	return &cfg, nil
}

func logLevelHookFunction(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t != reflect.TypeOf(slog.Level(0)) {
		return data, nil
	}

	if f.Kind() == reflect.String {
		var level slog.Level

		levelStr, ok := data.(string)
		if !ok {
			return nil, ErrInvalidLogLevel
		}

		err := level.UnmarshalText([]byte(levelStr))
		if err != nil {
			return nil, fmt.Errorf("unmarshal log level: %w", err)
		}

		return level, nil
	}

	return nil, &UnsupportedTypeError{
		Type: f.Kind().String(),
	}
}

func uuidHookFunction(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t != reflect.TypeOf(uuid.UUID{}) {
		return data, nil
	}

	if f.Kind() == reflect.String {
		uuidStr, ok := data.(string)
		if !ok {
			return nil, ErrInvalidUUID
		}

		id, err := uuid.Parse(uuidStr)
		if err != nil {
			return nil, fmt.Errorf("parse UUID: %w", err)
		}

		return id, nil
	}

	return nil, &UnsupportedTypeError{
		Type: f.Kind().String(),
	}
}
