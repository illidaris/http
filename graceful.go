package http

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

var (
	DefaultGracefulOption *GracefulOption = &GracefulOption{
		ShutTimeout: 3 * time.Second,
	}
)

type GracefulOption struct {
	ShutTimeout time.Duration
}

func Graceful(ip string, port int, h http.Handler, opt *GracefulOption) error {
	if opt == nil {
		opt = DefaultGracefulOption
	}
	// bind ip&port
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", ip, port),
		Handler: h,
	}
	errCh := make(chan error, 1)
	defer close(errCh)
	// listen
	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case s := <-ctx.Done():
		stop()
		timeoutCtx, cancel := context.WithTimeout(context.Background(), opt.ShutTimeout)
		defer cancel()
		if err := srv.Shutdown(timeoutCtx); err != nil {
			return fmt.Errorf("receive sign(%s),shut down timeout %f s", s, opt.ShutTimeout.Seconds())
		}
		return fmt.Errorf("receive sign(%s), shut down", s)
	case e := <-errCh:
		return fmt.Errorf("listen receive error %s", e)
	}
}
