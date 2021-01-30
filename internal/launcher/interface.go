package launcher

type Launcher interface {
	IsRunning() bool
	GetServerAddress() string
	Launch() error
	Shutdown() error
}
