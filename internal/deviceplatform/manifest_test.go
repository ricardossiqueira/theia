package deviceplatform

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ricardossiqueira/iot-device-core-go/manifest"
)

// TestManifestNeverEmitsNullSubscribe is a regression test: a nil (zero-value)
// MQTT.Subscribe slice marshals to "subscribe":null, which violates the
// manifest schema (subscribe must be an array) and crashes gateway-web's
// DeviceInterface, which calls .flatMap() on it unconditionally. See the
// comment on Manifest()'s Subscribe field.
func TestManifestNeverEmitsNullSubscribe(t *testing.T) {
	b, err := json.Marshal(Manifest())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"subscribe":null`) {
		t.Fatalf("manifest JSON contains \"subscribe\":null: %s", b)
	}
	if !strings.Contains(string(b), `"subscribe":[]`) {
		t.Fatalf("manifest JSON does not declare an empty subscribe array: %s", b)
	}
}

func TestManifestParsesAndCanonicalizes(t *testing.T) {
	b, err := json.Marshal(Manifest())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := manifest.Parse(string(b)); err != nil {
		t.Fatalf("manifest.Parse() on deviceplatform.Manifest(): %v", err)
	}
}
