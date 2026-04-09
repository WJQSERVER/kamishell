package make

import (
	"strconv"
	"testing"

	"kamishell/core"
)

func BenchmarkSnapshotBuildEnv(b *testing.B) {
	env := core.NewScriptEnvironment(core.NewEmptyEnvironment())
	for i := range 64 {
		env.SetPackageValue("env", "KEY_"+strconv.Itoa(i), strconv.Itoa(i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snapshot := snapshotBuildEnv(env)
		if len(snapshot) < 64 {
			b.Fatalf("unexpected snapshot size: %d", len(snapshot))
		}
	}
}

func BenchmarkNewBuildCommand(b *testing.B) {
	target := &Target{
		Name:    "kami",
		Sources: []string{"main.go", "completer.go"},
		BuildEnv: map[string]string{
			"GOOS":        "linux",
			"GOARCH":      "amd64",
			"CGO_ENABLED": "0",
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := newBuildCommand(target)
		if len(cmd.Args) == 0 {
			b.Fatal("expected build command args")
		}
	}
}
