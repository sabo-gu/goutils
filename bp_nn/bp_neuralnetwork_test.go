package bp_nn

import (
	"testing"

	deep "github.com/patrikeh/go-deep"

	"github.com/DoOR-Team/goutils/log"
)

func TestNewNNModel(t *testing.T) {
	nn := NewNNModel(2, []int{2, 4, 2}, deep.ActivationLinear, deep.ModeMultiClass,
		true, deep.NewNormal(1.0, 0),
	)
	log.Println(nn)
}
