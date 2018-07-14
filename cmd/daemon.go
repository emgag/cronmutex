package cmd

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emgag/cronmutex/internal/lib/config"
	"github.com/emgag/cronmutex/internal/lib/task"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(daemonCmd)
}

var daemonCmd = &cobra.Command{
	Use:   "daemon <cron.yml>",
	Short: "Cron daemon mode",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		options := config.Options{}
		err := viper.Unmarshal(&options)

		if err != nil {
			log.Print(err)
			return
		}

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	exit:
		for {
			f, err := ioutil.ReadFile(args[0])

			if err != nil {
				log.Printf("Failed loading config: %v\n", err)
				time.Sleep(10 * time.Second)
				continue
			}

			cron, err := task.NewCron(f, &options)

			if err != nil {
				log.Fatal(err)
			}

			cron.Start()

			s := <-sigs

			switch s {
			case syscall.SIGHUP:
				log.Printf("Reloading config")
				continue
			case syscall.SIGINT:
				fallthrough
			case syscall.SIGTERM:
				log.Printf("Terminating")
				break exit
			}

			cron.Stop()
		}
	},
}
