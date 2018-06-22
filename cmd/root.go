package cmd

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/emgag/cronmutex/internal/lib/config"
	"github.com/emgag/cronmutex/internal/lib/redis"
	"github.com/emgag/cronmutex/internal/lib/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/redsync.v1"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "cronmutex [flags] MUTEX-NAME COMMAND",
	Short: "Redis-backed mutex tool to prevent running commands on multiple machines.",
	Args: func(cmd *cobra.Command, args []string) error {
		if v, _ := cmd.Flags().GetBool("version"); !v && len(args) < 2 {
			return errors.New("Requires at least mutex name and a command")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Printf("cronmutex %s -- %s\n", version.Version, version.Commit)
			return
		}

		options := config.Options{}
		err := viper.Unmarshal(&options)

		if err != nil {
			log.Print(err)
			return
		}

		verbose, _ := cmd.Flags().GetBool("verbose")

		// exit code handling
		exitCode := 0
		defer func() {
			if verbose {
				log.Printf("Terminating")
			}

			os.Exit(exitCode)
		}()

		if wait, err := cmd.Flags().GetInt32("random-wait"); err == nil && wait > 0 {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			waitms := r.Int31n(wait * 1000)

			if verbose {
				log.Printf("Waiting for %dms", waitms)
			}

			time.Sleep(time.Duration(waitms) * time.Millisecond)
		}

		pool := redis.NewRedisConn(options)
		rs := redsync.New([]redsync.Pool{pool})

		mutexName := args[0]

		if options.Mutex.Prefix != "" {
			mutexName = options.Mutex.Prefix + mutexName
		}

		if verbose {
			log.Printf("Using %s as mutex name", mutexName)
		}

		mutexTTL := time.Duration(options.Mutex.DefaultTTL) * time.Second

		if ttl, err := cmd.Flags().GetInt("mutex-ttl"); err == nil && ttl > 0 {
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

		doUnlock := true

		defer func() {
			if !doUnlock {
				return
			}

			if verbose {
				log.Printf("Removing mutex again")
			}

			mutex.Unlock()
		}()

		if verbose {
			log.Printf("Running command: %s", strings.Join(args[1:], " "))
		}

		ex := exec.Command(args[1], args[2:]...)

		if n, err := cmd.Flags().GetBool("noout"); err != nil || !n {
			// pipe command stdout to main stdout
			stdout, err := ex.StdoutPipe()

			if err != nil {
				log.Print(err)
				return
			}

			go func() {
				io.Copy(os.Stdout, stdout)
			}()

			// pipe command stderr to main stderr
			stderr, err := ex.StderrPipe()

			if err != nil {
				log.Print(err)
				return
			}

			go func() {
				io.Copy(os.Stderr, stderr)
			}()
		}

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

		// check ttl to extend if command runs for longer
		extend := time.Tick(mutexTTL - 250*time.Millisecond)

		// command timeout channel
		timeout := time.After(cmdTTL)

		// clear timeout channel if no ttl for command is set
		if cmdTTL == 0 {
			timeout = nil
		}

		// signal handling
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			sig := <-sigs
			if verbose {
				log.Printf("Received signal %v", sig)
			}
			done <- nil
		}()

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

			case <-extend:
				// lock ttl expiring & still running, extend TTL?
				//
				// clear extend channel if no extending required and don't release lock at the end
				if ff, err := cmd.Flags().GetBool("fire-n-forget"); err == nil && ff {
					log.Println("Mutex is expiring, not renewing")
					extend = nil
					doUnlock = false
					continue
				}

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
		os.Exit(1)
	}
}

// init does actually initialize cli processing
func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is /etc/cronmutex.yml)")

	rootCmd.Flags().BoolP("fire-n-forget", "f", false, "Don't hold (extend) the lock while the command is running")
	rootCmd.Flags().IntP("mutex-ttl", "m", 0, "The TTL of the lock in X seconds")
	rootCmd.Flags().BoolP("noout", "n", false, "Don't dump STDOUT and STDERR from command")
	rootCmd.Flags().Int32P("random-wait", "w", 0, "Wait for a random duration between 0 and X seconds before acquiring the lock and starting the command")
	rootCmd.Flags().IntP("ttl", "t", 0, "Kill command after X seconds. Default is to wait until the command finishes by itself")
	rootCmd.Flags().BoolP("verbose", "v", false, "Tell what's happening with cronmutex")
	rootCmd.Flags().Bool("version", false, "Print version and exit")
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
