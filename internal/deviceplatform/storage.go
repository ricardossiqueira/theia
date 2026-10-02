package deviceplatform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ricardossiqueira/iot-device-core-go/platform"
)

// FileStorage persists the two device-local documents the core needs as
// plain files under Dir: identity.json (device_uid + Ed25519 private key)
// and provisioning.json (device_id + MQTT credentials). Both are written
// atomically (temp file + rename) and kept at 0600.
type FileStorage struct{ Dir string }

func (s FileStorage) LoadIdentity() ([]byte, error)     { return load(s.path("identity.json")) }
func (s FileStorage) SaveIdentity(v []byte) error       { return save(s.path("identity.json"), v) }
func (s FileStorage) LoadProvisioning() ([]byte, error) { return load(s.path("provisioning.json")) }
func (s FileStorage) SaveProvisioning(v []byte) error   { return save(s.path("provisioning.json"), v) }
func (s FileStorage) ClearProvisioning() error {
	if err := os.Remove(s.path("provisioning.json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear provisioning: %w", err)
	}
	return nil
}

func (s FileStorage) path(name string) string { return filepath.Join(s.Dir, name) }

func load(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, platform.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return b, nil
}

func save(path string, v []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod temp file for %s: %w", path, err)
	}
	if _, err := tmp.Write(v); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file for %s: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename into %s: %w", path, err)
	}
	return nil
}
