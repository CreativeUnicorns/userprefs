package userprefs

import (
	"testing"
)

func TestInterfaces(t *testing.T) {
	t.Name()
	var _ Storage = NewMockStorage()
	var _ Cache = NewMockCache()
	var _ Logger = &MockLogger{}
}
