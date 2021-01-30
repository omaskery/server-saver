package proxy

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/go-logr/logr"
	"github.com/taskcluster/slugid-go/slugid"
	"golang.org/x/sync/errgroup"
)

type ConnInfo interface {
	Uid() string
}

type ConnectHandler func(info ConnInfo)
type DisconnectHandler func(info ConnInfo)

type TargetProvider interface {
	GetServerAddress() string
}

type TargetProviderString string

func (t TargetProviderString) GetServerAddress() string {
	return string(t)
}

type Server struct {
	OnConnect    ConnectHandler
	OnDisconnect DisconnectHandler
	Target       TargetProvider
}

type proxyConnection struct {
	uid          string
	conn         net.Conn
	onConnect    ConnectHandler
	onDisconnect DisconnectHandler
}

func (p *proxyConnection) Uid() string {
	return p.uid
}

var _ ConnInfo = (*proxyConnection)(nil)

func (s *Server) Proxy(ctx context.Context, logger logr.Logger, bindAddr string) error {
	logger.Info("starting proxy server")
	defer logger.Info("proxy server exiting")

	l, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return fmt.Errorf("unable to listen: %w", err)
	}

	go func() {
		<-ctx.Done()
		logger.Info("context cancellation received, gracefully stopping")
		if err := l.Close(); err != nil {
			logger.Error(err, "error closing listener")
		}
	}()

	for {
		conn, err := l.Accept()
		if ctx.Err() != nil {
			return nil
		} else if err != nil {
			return fmt.Errorf("error accepting connections: %w", err)
		}

		c := &proxyConnection{
			uid:          slugid.Nice(),
			conn:         conn,
			onConnect:    s.OnConnect,
			onDisconnect: s.OnDisconnect,
		}
		go func() {
			if err := c.Handle(ctx, logger.WithName("conn"), s.Target.GetServerAddress()); err != nil {
				logger.Error(err, "error during proxied connection", "raddr", conn.RemoteAddr())
			}
		}()
	}
}

func (p *proxyConnection) Handle(ctx context.Context, logger logr.Logger, targetAddr string) error {
	l := logger.WithValues("raddr", p.conn.RemoteAddr(), "uid", p.uid)
	defer func() {
		if err := p.conn.Close(); err != nil {
			l.Error(err, "error closing incoming connection")
		}
	}()

	l.Info("connection established")
	defer l.Info("connection closed")

	if p.onConnect != nil {
		p.onConnect(p)
	}
	if p.onDisconnect != nil {
		defer p.onDisconnect(p)
	}

	target, err := net.Dial("tcp", targetAddr)
	if err != nil {
		return fmt.Errorf("error dialing proxy target: %w", err)
	}

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return proxyPipe(p.conn, target)
	})
	group.Go(func() error {
		return proxyPipe(target, p.conn)
	})

	return group.Wait()
}

func proxyPipe(src io.Reader, dst io.Writer) error {
	buffer := make([]byte, 64*1024)
	for {
		n, err := src.Read(buffer)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("error reading: %w", err)
		}
		b := buffer[:n]

		for len(b) > 0 {
			n, err = dst.Write(b)
			if err != nil {
				return fmt.Errorf("error writing: %w", err)
			}

			b = b[n:]
		}
	}
}
