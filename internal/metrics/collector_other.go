//go:build !linux

package metrics

import "fmt"

func NewLinuxCollector() (Collector, error) {
	return nil, fmt.Errorf("Orange Pi metrics collection requires Linux")
}
