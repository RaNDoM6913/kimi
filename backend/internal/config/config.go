package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Env      string         `yaml:"env"`
	HTTP     HTTPConfig     `yaml:"http"`
	Log      LogConfig      `yaml:"log"`
	Postgres PostgresConfig `yaml:"postgres"`
	Redis    RedisConfig    `yaml:"redis"`
	S3       S3Config       `yaml:"s3"`
	Auth     AuthConfig     `yaml:"auth"`
	Bot      BotConfig      `yaml:"bot"`
	Remote   RemoteConfig   `yaml:"remote"`
}

type HTTPConfig struct {
	Addr         string        `yaml:"addr"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

type PostgresConfig struct {
	DSN string `yaml:"dsn"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type S3Config struct {
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	UseSSL    bool   `yaml:"use_ssl"`
}

type AuthConfig struct {
	JWTSecret    string        `yaml:"jwt_secret"`
	JWTAccessTTL time.Duration `yaml:"jwt_access_ttl"`
	RefreshTTL   time.Duration `yaml:"refresh_ttl"`
}

type BotConfig struct {
	Token           string        `yaml:"token"`
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
	CircleRetention time.Duration `yaml:"circle_retention"`
}

type RemoteConfig struct {
	Limits     LimitsConfig     `yaml:"limits"`
	AntiAbuse  AntiAbuseConfig  `yaml:"antiabuse"`
	AdsInject  AdsInjectConfig  `yaml:"ads_inject"`
	Filters    FiltersConfig    `yaml:"filters"`
	GoalsMode  string           `yaml:"goals_mode"`
	Boost      BoostConfig      `yaml:"boost"`
	Cities     []CityConfig     `yaml:"cities"`
	MeDefaults MeDefaultsConfig `yaml:"me_defaults"`
}

type AntiAbuseConfig struct {
	LikeMaxPerSec        int     `yaml:"like_max_per_sec"`
	LikeMax10Sec         int     `yaml:"like_max_10s"`
	LikeMaxPerMin        int     `yaml:"like_max_min"`
	MinCardViewMS        int     `yaml:"min_card_view_ms"`
	RiskDecayHours       int     `yaml:"risk_decay_hours"`
	CooldownStepsSec     []int   `yaml:"cooldown_steps_sec"`
	ShadowThreshold      int     `yaml:"shadow_threshold"`
	ShadowRankMultiplier float64 `yaml:"shadow_rank_multiplier"`
	SuspectLikeThreshold int     `yaml:"suspect_like_threshold"`
}

type LimitsConfig struct {
	FreeLikesPerDay      int  `yaml:"free_likes_per_day"`
	PlusUnlimitedUI      bool `yaml:"plus_unlimited_ui"`
	PlusRatePerMinute    int  `yaml:"plus_rate_per_minute"`
	PlusRatePer10Seconds int  `yaml:"plus_rate_per_10sec"`
	PlusRewindsPerDay    int  `yaml:"plus_rewinds_per_day"`
}

type AdsInjectConfig struct {
	FreeEvery int `yaml:"free"`
	PlusEvery int `yaml:"plus"`
}

type FiltersConfig struct {
	AgeMin          int `yaml:"age_min"`
	AgeMax          int `yaml:"age_max"`
	RadiusDefaultKM int `yaml:"radius_default_km"`
	RadiusMaxKM     int `yaml:"radius_max_km"`
}

type BoostConfig struct {
	Duration time.Duration `yaml:"duration"`
}

type CityConfig struct {
	ID   string  `yaml:"id"`
	Name string  `yaml:"name"`
	Lat  float64 `yaml:"lat"`
	Lon  float64 `yaml:"lon"`
}

type MeDefaultsConfig struct {
	UsernamePrefix       string                 `yaml:"username_prefix"`
	CityID               string                 `yaml:"city_id"`
	ModerationStatus     string                 `yaml:"moderation_status"`
	Timezone             string                 `yaml:"timezone"`
	IsPlus               bool                   `yaml:"is_plus"`
	PlusDuration         time.Duration          `yaml:"plus_duration"`
	TooFastRetryAfterSec int64                  `yaml:"too_fast_retry_after_sec"`
	Entitlements         MeEntitlementsDefaults `yaml:"entitlements"`
}

type MeEntitlementsDefaults struct {
	SuperLikeCredits      int           `yaml:"superlike_credits"`
	BoostCredits          int           `yaml:"boost_credits"`
	RevealCredits         int           `yaml:"reveal_credits"`
	MessageWoMatchCredits int           `yaml:"message_wo_match_credits"`
	IncognitoDuration     time.Duration `yaml:"incognito_duration"`
}

func Default() Config {
	return Config{
		Env: "dev",
		HTTP: HTTPConfig{
			Addr:         ":8080",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
		},
		Log: LogConfig{Level: "debug"},
		Postgres: PostgresConfig{
			DSN: "postgres://app:app@localhost:5432/tgapp?sslmode=disable",
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
			DB:   0,
		},
		S3: S3Config{
			Endpoint:  "localhost:9000",
			AccessKey: "minio",
			SecretKey: "minio123",
			Bucket:    "tgapp-private",
			UseSSL:    false,
		},
		Auth: AuthConfig{
			JWTSecret:    "change-me",
			JWTAccessTTL: 15 * time.Minute,
			RefreshTTL:   720 * time.Hour,
		},
		Bot: BotConfig{
			Token:           "",
			CleanupInterval: 6 * time.Hour,
			CircleRetention: 365 * 24 * time.Hour,
		},
		Remote: RemoteConfig{
			Limits: LimitsConfig{
				FreeLikesPerDay:      35,
				PlusUnlimitedUI:      true,
				PlusRatePerMinute:    60,
				PlusRatePer10Seconds: 15,
				PlusRewindsPerDay:    3,
			},
			AntiAbuse: AntiAbuseConfig{
				LikeMaxPerSec:        2,
				LikeMax10Sec:         12,
				LikeMaxPerMin:        45,
				MinCardViewMS:        700,
				RiskDecayHours:       6,
				CooldownStepsSec:     []int{30, 60, 300, 1800, 86400},
				ShadowThreshold:      5,
				ShadowRankMultiplier: 0.4,
				SuspectLikeThreshold: 8,
			},
			AdsInject: AdsInjectConfig{
				FreeEvery: 7,
				PlusEvery: 37,
			},
			Filters: FiltersConfig{
				AgeMin:          18,
				AgeMax:          30,
				RadiusDefaultKM: 3,
				RadiusMaxKM:     50,
			},
			GoalsMode: "soft_priority",
			Boost: BoostConfig{
				Duration: 30 * time.Minute,
			},
			Cities: []CityConfig{
				{ID: "minsk", Name: "Minsk", Lat: 53.9006, Lon: 27.5590},
				{ID: "brest", Name: "Brest", Lat: 52.0976, Lon: 23.7341},
				{ID: "vitebsk", Name: "Vitebsk", Lat: 55.1904, Lon: 30.2049},
				{ID: "gomel", Name: "Gomel", Lat: 52.4412, Lon: 30.9878},
				{ID: "grodno", Name: "Grodno", Lat: 53.6694, Lon: 23.8131},
				{ID: "mogilev", Name: "Mogilev", Lat: 53.8980, Lon: 30.3325},
			},
			MeDefaults: MeDefaultsConfig{
				UsernamePrefix:       "tg_",
				CityID:               "minsk",
				ModerationStatus:     "pending",
				Timezone:             "Europe/Minsk",
				IsPlus:               false,
				PlusDuration:         0,
				TooFastRetryAfterSec: 0,
				Entitlements: MeEntitlementsDefaults{
					SuperLikeCredits:      0,
					BoostCredits:          0,
					RevealCredits:         0,
					MessageWoMatchCredits: 0,
					IncognitoDuration:     0,
				},
			},
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()

	if path != "" {
		if err := loadFromYAML(path, &cfg); err != nil {
			return Config{}, err
		}
	}

	if err := applyEnvOverrides(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func loadFromYAML(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("unmarshal config yaml: %w", err)
	}

	return nil
}

func applyEnvOverrides(cfg *Config) error {
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.Env = v
	}

	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTP.Addr = v
	}
	if err := overrideDuration("HTTP_READ_TIMEOUT", &cfg.HTTP.ReadTimeout); err != nil {
		return err
	}
	if err := overrideDuration("HTTP_WRITE_TIMEOUT", &cfg.HTTP.WriteTimeout); err != nil {
		return err
	}
	if err := overrideDuration("HTTP_IDLE_TIMEOUT", &cfg.HTTP.IdleTimeout); err != nil {
		return err
	}

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}

	if v := os.Getenv("POSTGRES_DSN"); v != "" {
		cfg.Postgres.DSN = v
	}

	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if err := overrideInt("REDIS_DB", &cfg.Redis.DB); err != nil {
		return err
	}

	if v := os.Getenv("S3_ENDPOINT"); v != "" {
		cfg.S3.Endpoint = v
	}
	if v := os.Getenv("S3_ACCESS_KEY"); v != "" {
		cfg.S3.AccessKey = v
	}
	if v := os.Getenv("S3_SECRET_KEY"); v != "" {
		cfg.S3.SecretKey = v
	}
	if v := os.Getenv("S3_BUCKET"); v != "" {
		cfg.S3.Bucket = v
	}
	if err := overrideBool("S3_USE_SSL", &cfg.S3.UseSSL); err != nil {
		return err
	}

	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if err := overrideDuration("JWT_ACCESS_TTL", &cfg.Auth.JWTAccessTTL); err != nil {
		return err
	}
	if err := overrideDuration("REFRESH_TTL", &cfg.Auth.RefreshTTL); err != nil {
		return err
	}
	if v := os.Getenv("BOT_TOKEN"); v != "" {
		cfg.Bot.Token = v
	}
	if err := overrideDuration("BOT_CLEANUP_INTERVAL", &cfg.Bot.CleanupInterval); err != nil {
		return err
	}
	if err := overrideDuration("BOT_CIRCLE_RETENTION", &cfg.Bot.CircleRetention); err != nil {
		return err
	}

	return nil
}

func overrideDuration(key string, target *time.Duration) error {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fmt.Errorf("parse %s duration: %w", key, err)
	}
	*target = d
	return nil
}

func overrideInt(key string, target *int) error {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("parse %s int: %w", key, err)
	}
	*target = n
	return nil
}

func overrideBool(key string, target *bool) error {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fmt.Errorf("parse %s bool: %w", key, err)
	}
	*target = b
	return nil
}
