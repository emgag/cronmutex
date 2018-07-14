package cmd

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

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

		hup := make(chan os.Signal, 1)
		signal.Notify(hup, syscall.SIGHUP)

		term := make(chan os.Signal, 1)
		signal.Notify(term, syscall.SIGINT, syscall.SIGTERM)

		f, err := ioutil.ReadFile(args[0])

		if err != nil {
			log.Fatalf("Failed loading cron file: %v\n", err)
		}

		cron, err := task.NewCron(f, &options)

		if err != nil {
			log.Fatal(err)
		}

		cron.Start()

	exit:
		for {
			select {
			case <-hup:
				log.Printf("Reloading config")

				f, err := ioutil.ReadFile(args[0])

				if err != nil {
					log.Printf("Failed loading cron file: %v\n", err)
					continue
				}

				cron.Stop()

				cron, err := task.NewCron(f, &options)

				if err != nil {
					log.Fatal(err)
				}

				cron.Start()

			case <-term:
				log.Printf("Terminating")
				break exit
			}
		}
	},
}
