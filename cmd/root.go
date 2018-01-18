package cmd

import (
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/emgag/cronmutex/internal/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/redsync.v1"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "cronmutex [flags] MUTEX-NAME COMMAND",
	Short: "Distributed mutex to prevent running commands on multiple machines.",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		options := lib.Options{}
		err := viper.Unmarshal(&options)
		verbose, _ := cmd.Flags().GetBool("verbose")

		exitCode := 0
		defer func() {
			if verbose {
				log.Printf("Terminating")
			}

			os.Exit(exitCode)

		}()

		if err != nil {
			log.Print(err)
			return
		}

		if wait, err := cmd.Flags().GetInt32("random-wait"); err == nil && wait > 0 {
			waitms := rand.Int31n(wait * 1000)
			if verbose {
				log.Printf("Waiting for %dms", waitms)
			}
			time.Sleep(time.Duration(waitms) * time.Millisecond)
		}

		pool := lib.NewRedisConn(options)
		rs := redsync.New([]redsync.Pool{pool})

		mutexName := args[0]

		if options.Mutex.Prefix != "" {
			mutexName = options.Mutex.Prefix + mutexName
		}

		if verbose {
			log.Printf("Using %s as mutex name", mutexName)
		}

		mutexTTL := time.Duration(options.Mutex.DefaultTTL) * time.Second

		if ttl, err := cmd.Flags().GetInt("mutex-ttl"); err != nil && ttl > 0 {
			mutexTTL = time.Duration(ttl) * time.Second
		}

		if verbose {
			log.Printf("Mutex TTL is %ds", int64(mutexTTL/time.Second))
		}

		cmdTTL := 0 * time.Second

		if ttl, err := cmd.Flags().GetInt("ttl"); err == nil && ttl > 0 {
			cmdTTL = time.Duration(ttl) * time.Second
		}

		if verbose && cmdTTL > 0 {
			log.Printf("Command will be terminated after %ds", int64(cmdTTL/time.Second))
		}

		mutex := rs.NewMutex(mutexName, redsync.SetExpiry(mutexTTL), redsync.SetTries(1))

		if err := mutex.Lock(); err != nil {
			log.Print(err)
			return
		}

		defer func() {
			if verbose {
				log.Printf("Removing mutex again")
			}

			mutex.Unlock()
		}()

		if verbose {
			log.Printf("Running command: %s", strings.Join(args[1:], " "))
		}

		ex := exec.Command(args[1], args[2:]...)

		// pipe command stdout to main stdout
		stdout, err := ex.StdoutPipe()

		if err != nil {
			log.Print(err)
			return
		}

		go func() {
			io.Copy(os.Stdout, stdout)

			//if _, err := io.Copy(os.Stdout, stdout); err != nil {
			//	log.Print(err)
			//}
		}()

		// pipe command stderr to main stderr
		stderr, err := ex.StderrPipe()

		if err != nil {
			log.Print(err)
			return
		}

		go func() {
			io.Copy(os.Stderr, stderr)

			//if _, err := io.Copy(os.Stderr, stderr); err != nil {
			//	log.Print(err)
			//}
		}()

		// start command
		if err := ex.Start(); err != nil {
			log.Print(err)
			return
		}

		done := make(chan error)

		// poll for status updates
		go func() {
			done <- ex.Wait()
		}()

		tick := time.Tick(mutexTTL - 250*time.Millisecond)
		timeout := time.After(cmdTTL)

		// clear timeout channel if no ttl for command is set
		if cmdTTL == 0 {
			timeout = nil
		}

	status:
		for {
			select {
			case err := <-done:
				// exited
				if err != nil {
					log.Print(err)
				}

				break status

			case <-timeout:
				// timed out
				if verbose {
					log.Println("Timeout reached, terminating command")
				}

				ex.Process.Kill()

			case <-tick:
				// still running, extend TTL
				if verbose {
					log.Println("Extending TTL")
				}

				mutex.Extend()
			}

		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is /etc/cronmutex.yml)")

	rootCmd.Flags().Bool("fire-n-forget", false, "Don't hold (extend) the lock while the command is running")
	rootCmd.Flags().Int("mutex-ttl", 0, "The TTL of the lock in X seconds")
	rootCmd.Flags().Bool("noout", false, "Don't dump STDOUT and STDERR from command")
	rootCmd.Flags().Int32("random-wait", 0, "Wait for a random time between 0 and X seconds before acquiring the lock and starting the command")
	rootCmd.Flags().Int("ttl", 0, "Kill command after X seconds. Default is to wait until the command finishes by itself")
	rootCmd.Flags().Bool("verbose", false, "Tell what's happening with cronmutex")
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
		viper.AddConfigPath("$HOME/.cronmutex")
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
