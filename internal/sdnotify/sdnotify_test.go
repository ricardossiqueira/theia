package sdnotify

import (
	"net"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestReadyNoSocketConfigured(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "")
	if err := Ready(); err != nil {
		t.Fatalf("Ready() with no NOTIFY_SOCKET: %v", err)
	}
}

func TestReadySendsReadyDatagram(t *testing.T) {
	if runtime.GOOS == "windows" {
		// NOTIFY_SOCKET/unixgram is a Linux-only mechanism; this is the
		// target platform (the Orange Pi deploy, and GitHub's Ubuntu CI
		// runner) in any case.
		t.Skip("unixgram sockets are not supported on windows")
	}
	socketPath := filepath.Join(t.TempDir(), "notify.sock")
	listener, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: socketPath, Net: "unixgram"})
	if err != nil {
		t.Fatalf("ListenUnixgram: %v", err)
	}
	defer listener.Close()

	t.Setenv("NOTIFY_SOCKET", socketPath)
	if err := Ready(); err != nil {
		t.Fatalf("Ready(): %v", err)
	}

	buf := make([]byte, 64)
	listener.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := listener.Read(buf)
	if err != nil {
		t.Fatalf("reading notify datagram: %v", err)
	}
	if got := string(buf[:n]); got != "READY=1\n" {
		t.Fatalf("datagram = %q, want %q", got, "READY=1\n")
	}
}
