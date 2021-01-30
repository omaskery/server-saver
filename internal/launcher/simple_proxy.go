package launcher

type SimpleProxyConfig struct {
	TargetAddress string `json:"target_address"`
}

type SimpleProxy struct {
	cfg SimpleProxyConfig
}

func NewSimpleProxy(cfg SimpleProxyConfig) *SimpleProxy {
	return &SimpleProxy{
		cfg: cfg,
	}
}

func (s SimpleProxy) IsRunning() bool {
	return true
}

func (s SimpleProxy) GetServerAddress() string {
	return s.cfg.TargetAddress
}

func (s SimpleProxy) Launch() error {
	return nil
}

func (s SimpleProxy) Shutdown() error {
	return nil
}

var _ Launcher = (*SimpleProxy)(nil)
