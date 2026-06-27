package version

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	withVersion(t, "0.1.0", "unknown", "unknown", func() {
		require.Equal(t, "veta 0.1.0", String())
	})

	withVersion(t, "", "unknown", "unknown", func() {
		require.Equal(t, "veta dev", String())
	})
}

func TestDetailed(t *testing.T) {
	withVersion(t, "0.1.0", "abc123", "2026-06-27", func() {
		require.Equal(t, "veta 0.1.0 commit abc123 built 2026-06-27", Detailed())
	})

	withVersion(t, "dev", "unknown", "unknown", func() {
		require.Equal(t, "veta dev", Detailed())
	})
}

// withVersion temporarily replaces build metadata during a test.
func withVersion(t *testing.T, version, commit, date string, test func()) {
	t.Helper()
	previousVersion := Version
	previousCommit := Commit
	previousDate := Date
	Version = version
	Commit = commit
	Date = date
	defer func() {
		Version = previousVersion
		Commit = previousCommit
		Date = previousDate
	}()

	test()
}
