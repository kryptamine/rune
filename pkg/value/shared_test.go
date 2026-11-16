package value

import (
	"testing"
	"unsafe"
)

func TestContainersKeepSharedFlagAtOneOffset(t *testing.T) {
	for name, at := range map[string]uintptr{
		"Obj":     unsafe.Offsetof(Obj{}.shared),
		"HashMap": unsafe.Offsetof(HashMap{}.shared),
		"ByteBuf": unsafe.Offsetof(ByteBuf{}.shared),
	} {
		if at != containerShared {
			t.Errorf("%s keeps its shared flag at %d, not %d", name, at, containerShared)
		}
	}
}
