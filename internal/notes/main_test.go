package notes

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()
	if benchData.root != "" {
		benchData.cleanOnce.Do(func() { _ = os.RemoveAll(benchData.root) })
	}
	os.Exit(code)
}
