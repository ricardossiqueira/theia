package deviceplatform

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ricardossiqueira/athena-go/platform"
)

func TestFileStorageRoundTrip(t *testing.T) {
	s := FileStorage{Dir: t.TempDir()}

	if _, err := s.LoadIdentity(); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("LoadIdentity() before save = %v, want ErrNotFound", err)
	}
	if _, err := s.LoadProvisioning(); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("LoadProvisioning() before save = %v, want ErrNotFound", err)
	}

	if err := s.SaveIdentity([]byte(`{"device_uid":"x"}`)); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadIdentity()
	if err != nil || string(got) != `{"device_uid":"x"}` {
		t.Fatalf("LoadIdentity() = %q, %v", got, err)
	}

	if err := s.SaveProvisioning([]byte(`{"device_id":"d"}`)); err != nil {
		t.Fatal(err)
	}
	got, err = s.LoadProvisioning()
	if err != nil || string(got) != `{"device_id":"d"}` {
		t.Fatalf("LoadProvisioning() = %q, %v", got, err)
	}

	if err := s.ClearProvisioning(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadProvisioning(); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("LoadProvisioning() after clear = %v, want ErrNotFound", err)
	}
	// Identity is untouched by clearing provisioning.
	if got, err := s.LoadIdentity(); err != nil || string(got) != `{"device_uid":"x"}` {
		t.Fatalf("LoadIdentity() after clear = %q, %v", got, err)
	}
}

func TestFileStorageClearProvisioningIsIdempotent(t *testing.T) {
	s := FileStorage{Dir: t.TempDir()}
	if err := s.ClearProvisioning(); err != nil {
		t.Fatalf("ClearProvisioning() on missing file: %v", err)
	}
}

func TestFileStoragePermissionsAndNoLeftoverTemp(t *testing.T) {
	dir := t.TempDir()
	s := FileStorage{Dir: dir}
	if err := s.SaveIdentity([]byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "identity.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Windows has no POSIX permission bits (Stat always reports 0666/0444
	// depending only on the read-only attribute) - this device only ever
	// runs on Linux, so only assert the mode there.
	if runtime.GOOS == "linux" {
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Fatalf("identity.json mode = %o, want 0600", perm)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("directory has %d entries after save, want 1 (no leftover temp file): %v", len(entries), entries)
	}
}
