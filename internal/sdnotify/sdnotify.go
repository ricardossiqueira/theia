// Package sdnotify implements the minimal sender side of systemd's
// NOTIFY_SOCKET protocol (sd_notify(3)): a single datagram write, no
// dependency needed for that.
package sdnotify

import (
	"net"
	"os"
)

// Ready notifies systemd that the service has finished starting (connected
// to its dependencies and done its first unit of real work), so a
// Type=notify unit's `systemctl start`/`restart` only reports success once
// that happened. It is a no-op when $NOTIFY_SOCKET is unset, so it's always
// safe to call outside systemd too (e.g. `just run` locally).
func Ready() error {
	addr := os.Getenv("NOTIFY_SOCKET")
	if addr == "" {
		return nil
	}
	if addr[0] == '@' {
		// Linux abstract namespace socket: leading '@' stands in for a NUL.
		addr = "\x00" + addr[1:]
	}
	conn, err := net.Dial("unixgram", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write([]byte("READY=1\n"))
	return err
}
