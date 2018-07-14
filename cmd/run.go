package cmd

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/emgag/cronmutex/internal/lib/config"
	"github.com/emgag/cronmutex/internal/lib/redis"
	"github.com/emgag/cronmutex/internal/lib/task"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().BoolP("fire-n-forget", "f", false, "Don't hold (extend) the lock while the command is running")
	runCmd.Flags().IntP("mutex-ttl", "m", 0, "The TTL of the lock in X seconds")
	runCmd.Flags().BoolP("noout", "n", false, "Don't dump STDOUT and STDERR from command")
	runCmd.Flags().IntP("random-wait", "w", 0, "Wait for a random duration between 0 and X seconds before acquiring the lock and starting the command")
	runCmd.Flags().IntP("ttl", "t", 0, "Kill command after X seconds. Default is to wait until the command finishes by itself")
	runCmd.Flags().BoolP("verbose", "v", false, "Tell what's happening with cronmutex")
}

var runCmd = &cobra.Command{
	Use:   "run [flags] <mutex-name> <command>",
	Short: "Run command",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("Requires at least mutex name and a command")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		options := config.Options{}
		err := viper.Unmarshal(&options)

		if err != nil {
			log.Print(err)
			return
		}

		runner := task.NewRunner(args[1:])
		runner.MutexName = args[0]
		runner.MutexPrefix = options.Mutex.Prefix
		runner.RedisPool = redis.NewRedisConn(&options)

		if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
			go func() {
				io.Copy(os.Stdout, runner.LogBuffer.Reader)
			}()
		} else {
			go func() {
				io.Copy(ioutil.Discard, runner.LogBuffer.Reader)
			}()
		}

		if wait, err := cmd.Flags().GetInt("random-wait"); err == nil && wait > 0 {
			runner.RandomWait = wait
		}

		runner.MutexTTL = time.Duration(options.Mutex.DefaultTTL) * time.Second

		if ttl, err := cmd.Flags().GetInt("mutex-ttl"); err == nil && ttl > 0 {
			runner.MutexTTL = time.Duration(ttl) * time.Second
		}

		if ttl, err := cmd.Flags().GetInt("ttl"); err == nil && ttl > 0 {
			runner.TaskTTL = time.Duration(ttl) * time.Second
		}

		if n, err := cmd.Flags().GetBool("noout"); err != nil || !n {
			go func() {
				io.Copy(os.Stdout, runner.Stdout.Reader)
			}()

			go func() {
				io.Copy(os.Stderr, runner.Stderr.Reader)
			}()
		} else {
			go func() {
				io.Copy(ioutil.Discard, runner.Stdout.Reader)
			}()

			go func() {
				io.Copy(ioutil.Discard, runner.Stderr.Reader)
			}()
		}

		if ff, err := cmd.Flags().GetBool("fire-n-forget"); err == nil {
			runner.FireAndForget = ff
		}

		rc, _ := runner.Run()
		os.Exit(rc)
	},
}
