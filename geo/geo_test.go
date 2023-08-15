package geo

import (
	"testing"

	"github.com/DoOR-Team/goutils/log"
)

func TestPoint_Distance(t *testing.T) {
	src := NewPointFromLngLat(104.5061454, 36.025389)
	dest := NewPointFromLngLat(104.5151211, 36.02818948)
	log.Println(src.Distance(dest))
	dest = NewPointFromLngLat(104.5061454, 36.025389)
	src = NewPointFromLngLat(104.5151211, 36.02818948)
	log.Println(src.Distance(dest))
}
