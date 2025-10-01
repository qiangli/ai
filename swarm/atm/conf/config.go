package conf

import (
	"github.com/qiangli/ai/swarm/api"
)

var secrets api.SecretStore

// import (
// 	"fmt"
// 	"os"
// 	"time"

// 	"github.com/spf13/viper"
// )

// const LandingPage = "/"
// const AssistantSharePage = "/chat"

// const (
// 	DHNTEnvDev  = "dev"
// 	DHNTEnvProd = "prod"
// )

// // TODO validate the values
// func MustEnvVars() error {
// 	envVars := []string{
// 		"SENDGRID_API_KEY",
// 		"DB_PATH",
// 		"API_MASTER_KEY",
// 		"JWT_SECRET",
// 		"GITHUB_CLIENT_ID",
// 		"GITHUB_CLIENT_SECRET",
// 		"GOOGLE_CLIENT_ID",
// 		"GOOGLE_CLIENT_SECRET",
// 		"LINKEDIN_CLIENT_ID",
// 		"LINKEDIN_CLIENT_SECRET",
// 		"DO_FAAS_TOKEN",
// 	}

// 	for _, v := range envVars {
// 		if os.Getenv(v) == "" {
// 			return fmt.Errorf("environment variable not set: %s", v)
// 		}
// 	}
// 	return nil
// }

// type Config struct {
// 	// DHNT_ENV
// 	// dev | prod
// 	Env string `mapstructure:"DHNT_ENV"`

// 	// gin
// 	GinMode string `mapstructure:"GIN_MODE"`

// 	// server port
// 	// :PORT
// 	Port int `mapstructure:"PORT"`

// 	// verbose
// 	// DEBUG
// 	Debug bool `mapstructure:"DEBUG"`

// 	// configuration base dir
// 	// ./
// 	Base string `mapstructure:"DHNT_BASE"`
// 	// static assets directory relative to base
// 	WebDir string `mapstructure:"WEB_DIR"`

// 	//
// 	BaseUrl string `mapstructure:"BASE_URL"`
// 	HubUrl  string `mapstructure:"HUB_URL"`

// 	// notification
// 	SendgridApiKey string `mapstructure:"SENDGRID_API_KEY"`

// 	// db
// 	// dialect: sqlite/mysql/pg
// 	DBType string `mapstructure:"DB_TYPE"`
// 	// dsn: postgres/mysql path: qlite
// 	DBPath string `mapstructure:"DB_PATH"`

// 	// User api-key encryption key
// 	ApiMasterKey string `mapstructure:"API_MASTER_KEY"`

// 	// auth
// 	FrontEndOrigin string `mapstructure:"FRONTEND_ORIGIN"`

// 	// oauth2
// 	JWTTokenSecret string        `mapstructure:"JWT_SECRET"`
// 	TokenExpiresIn time.Duration `mapstructure:"TOKEN_EXPIRED_IN"`
// 	TokenMaxAge    int           `mapstructure:"TOKEN_MAXAGE"`
// 	TokenDomain    string        `mapstructure:"TOKEN_DOMAIN"`

// 	//
// 	// ---------
// 	//
// 	GoogleClientID         string `mapstructure:"GOOGLE_CLIENT_ID"`
// 	GoogleClientSecret     string `mapstructure:"GOOGLE_CLIENT_SECRET"`
// 	GoogleOAuthRedirectUrl string `mapstructure:"GOOGLE_REDIRECT_URL"`

// 	GitHubClientID         string `mapstructure:"GITHUB_CLIENT_ID"`
// 	GitHubClientSecret     string `mapstructure:"GITHUB_CLIENT_SECRET"`
// 	GitHubOAuthRedirectUrl string `mapstructure:"GITHUB_REDIRECT_URL"`

// 	LinkedInClientID         string `mapstructure:"LINKEDIN_CLIENT_ID"`
// 	LinkedInClientSecret     string `mapstructure:"LINKEDIN_CLIENT_SECRET"`
// 	LinkedInOAuthRedirectUrl string `mapstructure:"LINKEDIN_REDIRECT_URL"`

// 	FacebookClientID         string `mapstructure:"FACEBOOK_CLIENT_ID"`
// 	FacebookClientSecret     string `mapstructure:"FACEBOOK_CLIENT_SECRET"`
// 	FacebookOAuthRedirectUrl string `mapstructure:"FACEBOOK_REDIRECT_URL"`

// 	InstagramClientID         string `mapstructure:"INSTAGRAM_CLIENT_ID"`
// 	InstagramClientSecret     string `mapstructure:"INSTAGRAM_CLIENT_SECRET"`
// 	InstagramOAuthRedirectUrl string `mapstructure:"INSTAGRAM_REDIRECT_URL"`

// 	DiscordClientID         string `mapstructure:"DISCORD_CLIENT_ID"`
// 	DiscordClientSecret     string `mapstructure:"DISCORD_CLIENT_SECRET"`
// 	DiscordOAuthRedirectUrl string `mapstructure:"DISCORD_REDIRECT_URL"`

// 	LineClientID         string `mapstructure:"LINE_CLIENT_ID"`
// 	LineClientSecret     string `mapstructure:"LINE_CLIENT_SECRET"`
// 	LineOAuthRedirectUrl string `mapstructure:"LINE_REDIRECT_URL"`

// 	TwitterClientID         string `mapstructure:"TWITTER_CLIENT_ID"`
// 	TwitterClientSecret     string `mapstructure:"TWITTER_CLIENT_SECRET"`
// 	TwitterOAuthRedirectUrl string `mapstructure:"TWITTER_REDIRECT_URL"`

// 	TikTokClientID         string `mapstructure:"TIKTOK_CLIENT_ID"`
// 	TikTokClientSecret     string `mapstructure:"TIKTOK_CLIENT_SECRET"`
// 	TikTokOAuthRedirectUrl string `mapstructure:"TIKTOK_REDIRECT_URL"`

// 	//
// 	DoFaasUrl   string `mapstructure:"DO_FAAS_URL"`
// 	DoFaasToken string `mapstructure:"DO_FAAS_TOKEN"`
// }

// // base/.env
// // base/config.env
// func LoadConfig(configFile string) (*Config, error) {
// 	viper.SetConfigFile(configFile)
// 	viper.SetConfigType("env")

// 	viper.AutomaticEnv()

// 	if err := viper.ReadInConfig(); err != nil {
// 		return nil, err
// 	}

// 	var cfg Config
// 	if err := viper.Unmarshal(&cfg); err != nil {
// 		return nil, err
// 	}

// 	if len(cfg.ApiMasterKey) < 8 {
// 		return nil, fmt.Errorf("ApiMasterKey not seet or too short (<8)")
// 	}

// 	return &cfg, nil
// }
