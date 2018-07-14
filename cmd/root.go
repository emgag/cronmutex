package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "cronmutex [flags] <MUTEX-NAME> <COMMAND>",
	Short: "Redis-backed cron daemon and mutex tool to prevent running commands on multiple machines.",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// init does actually initialize cli processing
func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is /etc/cronmutex.yml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigName("cronmutex")

	// set defaults for redis
	viper.SetDefault("redis.uri", "redis://127.0.0.1:6379")

	// set defaults for varnish
	viper.SetDefault("mutex.prefix", "")
	viper.SetDefault("mutex.default_ttl", 300)

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("/etc")
		viper.AddConfigPath("$HOME/.config")
		viper.AddConfigPath(".")
	}

	viper.SetEnvPrefix("cm")
	viper.AutomaticEnv()

	// if a config file is found, read it in.
	err := viper.ReadInConfig()

	if err != nil {
		log.Fatal("Could not open config file.")
	}
}
