package director

import (
	"context"
	"github.com/go-logr/logr"
	"server-saver/internal/jsonutil"
	"server-saver/internal/launcher"
	"time"
)

const DefaultIdlePeriod = 5 * time.Minute

type Config struct {
	IdlePeriod jsonutil.Duration `json:"idle_period"`
}

type Director struct {
	logger            logr.Logger
	connections       map[string]*connection
	actions           chan<- func()
	launcher          launcher.Launcher
	scheduledShutdown *time.Time

	cfg Config
}

type connection struct {
	startTime time.Time
}

func New(ctx context.Context, logger logr.Logger, l launcher.Launcher, cfg Config) *Director {
	if cfg.IdlePeriod == jsonutil.Duration(0) {
		cfg.IdlePeriod = jsonutil.Duration(DefaultIdlePeriod)
	}

	actions := make(chan func())
	d := &Director{
		logger:      logger,
		connections: map[string]*connection{},
		actions:     actions,
		launcher:    l,
		cfg:         cfg,
	}
	go d.run(ctx, actions)
	return d
}

func (d *Director) RegisterConnection(uid string) {
	connection := &connection{
		startTime: time.Now(),
	}

	d.actions <- func() {
		d.connections[uid] = connection
		d.logger.Info("registered connection", "uid", uid, "count", len(d.connections))

		if d.scheduledShutdown != nil {
			d.logger.Info("scheduled shutdown cancelled due to new connection")
			d.scheduledShutdown = nil
		}

		if !d.launcher.IsRunning() {
			d.logger.Info("server not currently running, launching server")
			if err := d.launcher.Launch(); err != nil {
				d.logger.Error(err, "failed to launch server", err)
			}
		}
	}
}

func (d *Director) UnregisterConnection(uid string) {
	disconnectTime := time.Now()

	d.actions <- func() {
		l := d.logger.WithValues(
			"uid", uid,
			"count", len(d.connections),
		)

		connection, ok := d.connections[uid]
		if !ok {
			l.Info("unknown unregistration attempt")
			return
		}

		delete(d.connections, uid)
		duration := disconnectTime.Sub(connection.startTime)
		l.Info("unregistered connection", "duration", duration, "count", len(d.connections))

		if len(d.connections) == 0 {
			d.scheduleServerShutdown()
		}
	}
}

func (d *Director) scheduleServerShutdown() {
	idlePeriod := time.Duration(d.cfg.IdlePeriod)

	shutdownTime := time.Now().Add(idlePeriod)
	d.scheduledShutdown = &shutdownTime

	d.logger.Info("server empty, scheduling shutdown", "scheduled-time", *d.scheduledShutdown, "idle-period", idlePeriod)
	time.AfterFunc(idlePeriod, func() {
		d.actions <- func() {
			if d.scheduledShutdown == nil {
				d.logger.V(1).Info("scheduled shutdown did nothing: aborted")
				return
			}

			if d.scheduledShutdown.After(time.Now()) {
				d.logger.V(1).Info("scheduled shutdown did nothing: pushed back", "scheduled-time", *d.scheduledShutdown)
				return
			}

			d.logger.Info("server empty for idle period, performing shutdown", "idle-period", idlePeriod)
			if err := d.launcher.Shutdown(); err != nil {
				d.logger.Error(err, "error shutting down server")
			}
		}
	})
}

func (d *Director) run(ctx context.Context, actions <-chan func()) {
	d.logger.Info("director actor task running")
	defer d.logger.Info("director actor task exiting")

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("director received context cancellation")
			return
		case action, more := <-actions:
			if !more {
				return
			}

			action()
		}
	}
}
