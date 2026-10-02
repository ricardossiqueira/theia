package deviceplatform

import (
	"fmt"

	"github.com/grandcat/zeroconf"
)

// Announcer is the subset of *runtime.Runtime.NetworkReady's result needed
// to register the mDNS service. Matching iot-gateway's devicev2.Announcement
// decoding (announcementFromMDNS): TXT entries are "key=value" strings.
type Announcement struct {
	Service string
	Port    uint16
	TXT     map[string]string
}

// Announce registers "_iot-device._tcp" via the same grandcat/zeroconf
// library iot-gateway uses to Browse for it. The registration stays active
// (and its TXT record reflects the device's current state) until Shutdown is
// called on the returned server.
func Announce(a Announcement) (*zeroconf.Server, error) {
	if a.Service == "" || a.Port == 0 {
		return nil, fmt.Errorf("announcement requires a service and port")
	}
	text := make([]string, 0, len(a.TXT))
	for k, v := range a.TXT {
		text = append(text, k+"="+v)
	}
	return zeroconf.Register(a.TXT["uid"], a.Service, "local.", int(a.Port), text, nil)
}
