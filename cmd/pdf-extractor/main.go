//go:build unix

package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	runner := pdfextract.Poppler{}
	check, cancel := context.WithTimeout(ctx, 5*time.Second)
	err := runner.Verify(check)
	cancel()
	if err != nil {
		log.Fatal("pinned PDF extractor unavailable")
	}
	const socket = "/run/grace-pdf/extractor.sock"
	if info, e := os.Lstat(socket); e == nil {
		if info.Mode()&os.ModeSocket == 0 {
			log.Fatal("invalid socket path")
		}
		if e = os.Remove(socket); e != nil {
			log.Fatal("cannot clean socket")
		}
	} else if !os.IsNotExist(e) {
		log.Fatal("cannot inspect socket")
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		log.Fatal("cannot listen on private socket")
	}
	defer listener.Close()
	if err = os.Chmod(socket, 0660); err != nil {
		log.Fatal("cannot protect socket")
	}
	server := &http.Server{Handler: pdfextract.NewServer(runner), ReadTimeout: 32 * time.Second, WriteTimeout: 35 * time.Second, IdleTimeout: 5 * time.Second, MaxHeaderBytes: 4096, BaseContext: func(net.Listener) context.Context { return ctx }}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	if err = server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatal("PDF helper stopped")
	}
}
