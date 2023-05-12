// Roughly copied from co2monz/common - should make generic
package tailscale

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"tailscale.com/client/tailscale"
	"tailscale.com/tsnet"
)

func eatLogs(msg string, _ ...any) {}

// Tailscale server instance
var S *tsnet.Server

// Tailscale local client instance
var LC *tailscale.LocalClient

// true if we're connected to a tailnet successfully
var Connected bool

// current tailnet DNS suffix
var DNSSuffix string

// Login to tailscale and initialize exported instanecs (S, LC, C)
func Init(tsname string, dir string, timeout time.Duration) error {
	log.Printf("Connecting to Tailscale as %s...", tsname)
	start := time.Now()

	logf := eatLogs
	if _, ok := os.LookupEnv("DEBUG_TS"); ok {
		logf = nil
	}

	S = &tsnet.Server{
		Dir:      dir,
		Hostname: tsname,
		Logf:     logf,
	}

	lc, err := S.LocalClient()
	if err != nil {
		errors.Wrap(err, "couldn't get Tailscale local client")
	}
	LC = lc

	stateStr := "unknown"
	msgDone := false
	for {
		status, err := LC.Status(context.Background())
		if err != nil {
			log.Printf("Couldn't get Tailscale status: %v", err)
		} else {
			stateStr = status.BackendState
			if stateStr == "Running" {
				log.Printf("Connected to Tailscale as %s (%s)", status.Self.DNSName, status.Self.TailscaleIPs[0])
				Connected = true
				DNSSuffix = status.CurrentTailnet.MagicDNSSuffix
				break
			}
		}
		if timeout != 0 && time.Since(start) > timeout {
			return fmt.Errorf("tailscale not running (currently %s) after %s", stateStr, timeout)
		} else if timeout == 0 && !msgDone && time.Since(start) > 30*time.Second {
			log.Printf("Tailscale did not connect after 30s, will continuing trying in background...")
			msgDone = true
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

func Serve(h http.Handler) error {
	ln, err := S.Listen("tcp", ":80")
	if err != nil {
		return err
	}

	log.Print("Ready to serve on Tailscale!")
	err = http.Serve(ln, h)
	// Not expecting to return, so mark ourselves as disconnected if/when it does
	Connected = false
	return err
}
