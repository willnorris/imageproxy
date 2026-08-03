// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

// Native OS-service support (fork addition). Lets the single imageproxy binary
// install/run itself as a Windows service (via the SCM — no nssm or other wrapper),
// a Linux systemd unit, or a macOS launchd job, and run in the foreground otherwise.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kardianos/service"
)

var svcAction = flag.String("service", "", "control the service and exit: install | uninstall | start | stop | restart")
var logFile = flag.String("logFile", "", "append logs to this file (recommended when running as a service, whose stdout is discarded)")

// program adapts the HTTP server to the service.Interface lifecycle.
type program struct {
	server *http.Server
	ln     net.Listener
}

// Start is non-blocking (required by service.Interface): it binds the listener and
// serves in a goroutine. Binding here (not at install time) means `-service install`
// never needs the port free.
func (p *program) Start(s service.Service) error {
	var err error
	if path, ok := strings.CutPrefix(p.server.Addr, "unix:"); ok {
		p.ln, err = net.Listen("unix", path)
	} else {
		p.ln, err = net.Listen("tcp", p.server.Addr)
	}
	if err != nil {
		return err
	}
	log.Printf("imageproxy listening on %s", p.server.Addr)
	go func() {
		if err := p.server.Serve(p.ln); err != nil && err != http.ErrServerClosed {
			log.Printf("serve error: %v", err)
		}
	}()
	return nil
}

// Stop gracefully drains in-flight requests.
func (p *program) Stop(s service.Service) error {
	if p.ln == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return p.server.Shutdown(ctx)
}

// serviceArgs returns the process args with the -service control flag (and its value)
// removed, so the installed service's registered command line carries the same runtime
// flags minus the one-shot control action.
func serviceArgs() []string {
	var out []string
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-service" || a == "--service" {
			i++ // also skip its value
			continue
		}
		if strings.HasPrefix(a, "-service=") || strings.HasPrefix(a, "--service=") {
			continue
		}
		out = append(out, a)
	}
	return out
}

// runWithService runs the server as a managed OS service when launched by the service
// manager, or in the foreground when run interactively. With -service <action> it
// performs the control action (install/uninstall/start/stop/restart) and exits.
func runWithService(server *http.Server) {
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			log.Fatalf("cannot open -logFile %q: %v", *logFile, err)
		}
		log.SetOutput(f)
	}

	prg := &program{server: server}
	cfg := &service.Config{
		Name:        "imageproxy",
		DisplayName: "imageproxy (WebP/AVIF image resizer)",
		Description: "On-the-fly image resizer for self-hosted /media (WebP/AVIF/JPEG).",
		Arguments:   serviceArgs(),
	}
	s, err := service.New(prg, cfg)
	if err != nil {
		log.Fatalf("service init: %v", err)
	}

	if *svcAction != "" {
		if err := service.Control(s, *svcAction); err != nil {
			log.Fatalf("service %q failed: %v (valid actions: %v)", *svcAction, err, service.ControlAction)
		}
		fmt.Printf("service %s: ok\n", *svcAction)
		return
	}

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
