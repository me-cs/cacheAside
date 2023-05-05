package cacheAside

import (
	"testing"
	"time"
)

func TestUnstable_AroundDuration(t *testing.T) {
	unstable := NewUnstable(0.05)
	for i := 0; i < 1000; i++ {
		val := unstable.AroundDuration(time.Second)
		if !(float64(time.Second)*0.95 <= float64(val)) {
			t.Fatal("val is not in range")
		}
		if !(float64(val) <= float64(time.Second)*1.05) {
			t.Fatal("val is not in range")
		}
	}
}
