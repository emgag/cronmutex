package task

import (
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-redsync/redsync"
	"github.com/gomodule/redigo/redis"
)

type Runner struct {
	Cmd []string

	MutexName   string
	MutexPrefix string
	MutexTTL    time.Duration

	TaskTTL time.Duration

	RandomWait int

	FireAndForget bool

	Stdout    RunnerPipe
	Stderr    RunnerPipe
	LogBuffer RunnerPipe

	RedisPool *redis.Pool
}

type RunnerPipe struct {
	Reader *io.PipeReader
	Writer *io.PipeWriter
}

func (r *Runner) Run() (int, error) {
	logger := log.New(r.LogBuffer.Writer, "["+r.MutexName+"] ", log.LstdFlags)

	if r.RandomWait > 0 {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		waitms := rnd.Int31n(int32(r.RandomWait) * 1000)
		logger.Printf("Waiting for %dms", waitms)
		time.Sleep(time.Duration(waitms) * time.Millisecond)
	}

	mutexName := r.MutexPrefix + r.MutexName
	logger.Printf("Using %s as mutex name", mutexName)

	logger.Printf("Mutex TTL is %v", r.MutexTTL)

	if r.TaskTTL > 0 {
		logger.Printf("Command will be terminated after %v", r.TaskTTL)
	}

	rs := redsync.New([]redsync.Pool{r.RedisPool})
	mutex := rs.NewMutex(mutexName, redsync.SetExpiry(r.MutexTTL), redsync.SetTries(1))

	if err := mutex.Lock(); err != nil {
		logger.Print(err)
		return 1, err
	}

	doUnlock := true

	defer func() {
		if !doUnlock {
			return
		}

		logger.Printf("Removing mutex again")

		mutex.Unlock()
	}()

	logger.Printf("Running command: %s", strings.Join(r.Cmd, " "))

	ex := exec.Command(r.Cmd[0], r.Cmd[1:]...)
	ex.Stdout = r.Stdout.Writer
	ex.Stderr = r.Stderr.Writer

	// start command
	if err := ex.Start(); err != nil {
		logger.Print(err)
		return 1, err
	}

	done := make(chan error)

	// poll for status updates
	go func() {
		done <- ex.Wait()
	}()

	// check ttl to extend if command runs for longer
	extend := time.Tick(r.MutexTTL - 250*time.Millisecond)

	// command timeout channel
	timeout := time.After(r.TaskTTL)

	// clear timeout channel if no ttl for command is set
	if r.TaskTTL == 0 {
		timeout = nil
	}

	// signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logger.Printf("Received signal %v", sig)
		done <- nil
	}()

status:
	for {
		select {
		case err := <-done:
			// exited
			if err != nil {
				logger.Print(err)
				return 1, err
			}

			break status

		case <-timeout:
			// timed out
			logger.Println("Timeout reached, terminating command")

			ex.Process.Kill()

		case <-extend:
			// lock ttl expiring & still running, extend TTL?
			//
			// clear extend channel if no extending required and don't release lock at the end
			if r.FireAndForget {
				logger.Println("Mutex is expiring, not renewing")
				extend = nil
				doUnlock = false
				continue
			}

			logger.Println("Extending TTL")

			mutex.Extend()
			extend = time.Tick(r.MutexTTL - 250*time.Millisecond)
		}
	}

	return 0, nil
}

func NewRunner(cmd []string) *Runner {
	return &Runner{
		Cmd:       cmd,
		LogBuffer: NewRunnerPipe(),
		Stderr:    NewRunnerPipe(),
		Stdout:    NewRunnerPipe(),
	}
}

func NewRunnerPipe() RunnerPipe {
	r, w := io.Pipe()

	return RunnerPipe{r, w}
}
