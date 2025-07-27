package hub

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/hub"
	"github.com/qiangli/ai/swarm/api"
)

var viper = internal.V

var startCmd = &cobra.Command{
	Use:                   "start",
	Short:                 "Start Hub service",
	DisableFlagsInUseLine: false,
	DisableSuggestions:    true,
	Run: func(cmd *cobra.Command, args []string) {
		var cfg = &api.AppConfig{}

		if err := internal.ParseConfig(viper, cfg, args); err != nil {
			internal.Exit(err)
		}
		parseStartFlags(cfg)
		if err := hub.StartServer(cfg); err != nil {
			internal.Exit(err)
		}
	},
}

func init() {
	addStartFlags(startCmd.Flags())
	startCmd.CompletionOptions.DisableDefaultCmd = true

	// Bind the flags to viper using underscores
	startCmd.Flags().VisitAll(func(f *pflag.Flag) {
		key := strings.ReplaceAll(f.Name, "-", "_")
		viper.BindPFlag(key, f)
	})

	//
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ai")
	viper.BindEnv("api-key", "AI_API_KEY", "OPENAI_API_KEY")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	HubCmd.AddCommand(startCmd)
}

func addStartFlags(flags *pflag.FlagSet) {
	flags.SortFlags = true

	// services
	flags.Bool("hub", true, "Start hub services")
	flags.String("address", ":58080", "Hub service host:port")
	flags.MarkHidden("hub")

	// the following will only start if hub server is enabled
	flags.Bool("pg", true, "Start postgres server")
	flags.String("pg-address", ":5432", "Postgres server host:port")

	flags.Bool("mysql", true, "Start mysql server")
	flags.String("mysql-address", ":3306", "MySQL server host:port")

	flags.Bool("redis", true, "Start redis server")
	flags.String("redis-address", ":6379", "Redis server host:port")

	flags.Bool("terminal", true, "Start web terminal server")
	flags.String("terminal-address", ":58088", "Web terminal server host:port")

	flags.Bool("llm-proxy", true, "Start LLM proxy server")
	flags.String("llm-proxy-address", ":8000", "LLM proxy server host:port")
	flags.String("llm-proxy-secret", "sk-secret", "LLM proxy server secret for local access")
	// TODO use values from models
	flags.String("llm-proxy-api-key", "", "OpenAI api key")
}

func parseStartFlags(app *api.AppConfig) {
	// Hub services
	hub := &api.HubConfig{}
	app.Hub = hub

	hub.Enable = viper.GetBool("hub")
	hub.Address = viper.GetString("address")
	hub.Pg = viper.GetBool("pg")
	hub.PgAddress = viper.GetString("pg_address")
	hub.Mysql = viper.GetBool("mysql")
	hub.MysqlAddress = viper.GetString("mysql_address")
	hub.Redis = viper.GetBool("redis")
	hub.RedisAddress = viper.GetString("redis_address")
	hub.Terminal = viper.GetBool("terminal")
	hub.TerminalAddress = viper.GetString("terminal_address")
	hub.LLMProxy = viper.GetBool("llm_proxy")
	hub.LLMProxyAddress = viper.GetString("llm_proxy_address")
	hub.LLMProxySecret = viper.GetString("llm_proxy_secret")
	hub.LLMProxyApiKey = viper.GetString("llm_proxy_api_key")
}
