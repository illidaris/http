package http

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func Graceful(ip string, port int, h http.Handler, shutTimeout time.Duration) error {
	if shutTimeout < time.Second*3 {
		shutTimeout = time.Second * 3
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
		timeoutCtx, cancel := context.WithTimeout(context.Background(), shutTimeout)
		defer cancel()
		if err := srv.Shutdown(timeoutCtx); err != nil {
			return fmt.Errorf("receive sign(%s),shut down timeout %f s", s, shutTimeout.Seconds())
		}
		return fmt.Errorf("receive sign(%s), shut down", s)
	case e := <-errCh:
		return fmt.Errorf("listen receive error %s", e)
	}
}
