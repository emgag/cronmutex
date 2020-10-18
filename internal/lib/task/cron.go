package task

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/emgag/cronmutex/internal/lib/config"
	"github.com/emgag/cronmutex/internal/lib/redis"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v2"
)

// CronEntry holds a single entry from the crontab YAML file
type CronEntry struct {
	Name    string   `yaml:"name"`
	Cron    string   `yaml:"cron"`
	Command []string `yaml:"command"`
	Options map[string]interface{}
}

// NewCron creates a new cron runner from config data
func NewCron(cfg []byte, options *config.Options) (*cron.Cron, error) {

	entries := []*CronEntry{}
	err := yaml.Unmarshal(cfg, &entries)

	if err != nil {
		return nil, err
	}

	c := cron.New()

	for _, e := range entries {
		log.Printf("Adding %s @ %s %v\n", e.Name, e.Cron, e.Command)
		entry := e

		c.AddFunc(e.Cron, func() {
			runner := NewRunner(entry.Command)
			runner.MutexName = entry.Name
			runner.MutexPrefix = options.Mutex.Prefix
			runner.RedisPool = redis.NewRedisConn(options)
			runner.MutexTTL = time.Duration(options.Mutex.DefaultTTL) * time.Second

			if mutexTTL, ok := entry.Options["mutexttl"]; ok {
				runner.TaskTTL = time.Second * time.Duration(mutexTTL.(int))
			}

			if ttl, ok := entry.Options["ttl"]; ok {
				runner.TaskTTL = time.Second * time.Duration(ttl.(int))
			}

			if wait, ok := entry.Options["randomwait"]; ok {
				runner.RandomWait = wait.(int)
			}

			if ff, ok := entry.Options["fireandforget"]; ok {
				runner.FireAndForget = ff.(bool)
			}

			go func() {
				io.Copy(os.Stdout, runner.LogBuffer.Reader)
			}()

			go func() {
				io.Copy(os.Stdout, runner.Stdout.Reader)
			}()

			go func() {
				io.Copy(os.Stderr, runner.Stderr.Reader)
			}()

			rc, err := runner.Run()

			if err != nil {
				log.Printf("Error running %s: %s", entry.Name, err)
			}

			log.Printf("Finished running %s, exit code %d", entry.Name, rc)
		})

	}

	return c, nil
}
