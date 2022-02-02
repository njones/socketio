package engineio

import "testing"

func TestServerV2Basic(t *testing.T) {
	a := NewServerV2()
	t.Error(a)
}
