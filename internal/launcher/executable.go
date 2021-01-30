package launcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"os/exec"
)

var (
	ErrAlreadyRunning = errors.New("instance of server is already running")
)

type ExecutableConfig struct {
	Path    string   `json:"path"`
	Args    []string `json:"args"`
	Cwd     string   `json:"cwd"`
	Address string   `json:"address"`
}

type Executable struct {
	logger     logr.Logger
	process    *exec.Cmd
	generation int
	cfg        ExecutableConfig

	actions chan<- func()
	ctx     context.Context
}

func NewExecutableLauncher(ctx context.Context, logger logr.Logger, cfg ExecutableConfig) *Executable {
	actions := make(chan func())
	e := &Executable{
		logger:  logger,
		ctx:     ctx,
		cfg:     cfg,
		actions: actions,
	}

	go e.run(ctx, actions)

	return e
}

func (e *Executable) IsRunning() bool {
	r := make(chan bool, 1)
	e.actions <- func() {
		r <- e.process != nil
	}
	return <-r
}

func (e *Executable) GetServerAddress() string {
	return e.cfg.Address
}

func (e *Executable) Launch() error {
	r := make(chan error, 1)

	launch := func() error {
		if e.process != nil {
			return ErrAlreadyRunning
		}

		e.generation += 1
		e.logger.Info("launching server process", "path", e.cfg.Path)
		e.process = exec.CommandContext(e.ctx, e.cfg.Path, e.cfg.Args...)
		e.process.Dir = e.cfg.Cwd
		if err := e.process.Start(); err != nil {
			return fmt.Errorf("error launching server process: %w", err)
		}

		go func() {
			defer func() {
				e.processCompleted()
			}()

			err := e.process.Wait()
			if err != nil {
				e.logger.Error(err, "error waiting on launched server process")
				return
			}
		}()

		return nil
	}

	e.actions <- func() {
		r <- launch()
	}

	return <-r
}

func (e *Executable) Shutdown() error {
	generation := e.getGeneration()
	r := make(chan error, 1)
	e.actions <- func() {
		if e.generation > generation {
			e.logger.V(1).Info("shutdown request from previous generation, ignoring", "request-generation", generation, "current-generation", e.generation)
			r <- nil
			return
		}

		if e.process == nil {
			r <- nil
			return
		}

		if err := e.process.Process.Kill(); err != nil {
			r <- fmt.Errorf("error killing launched server process: %w", err)
			return
		}

		r <- nil
	}
	return <-r
}

var _ Launcher = (*Executable)(nil)

func (e *Executable) run(ctx context.Context, actions <-chan func()) {
	e.logger.Info("executable actor task running")
	defer e.logger.Info("executable actor task exiting")

	for {
		select {
		case <-ctx.Done():
			return
		case action, more := <-actions:
			if !more {
				return
			}

			action()
		}
	}
}

func (e *Executable) processCompleted() {
	e.actions <- func() {
		e.logger.Info("finished running server process")
		e.process = nil
	}
}

func (e *Executable) getGeneration() int {
	r := make(chan int, 1)
	e.actions <- func() {
		r <- e.generation
	}
	return <-r
}
