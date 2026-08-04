package enabler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/skupperproject/skupper/pkg/nonkube/api"
	"gotest.tools/v3/assert"
)

func TestFilter(t *testing.T) {
	e, _ := newTestEnabler(t, 1000)
	scriptsSegment := "/" + string(api.ScriptsPath) + "/"

	cases := []struct {
		path string
		want bool
	}{
		{filepath.Join("/data/namespaces/west", string(api.ScriptsPath), "skupper-west.service"), true},
		{filepath.Join("/data/namespaces/west", string(api.ScriptsPath), "skupper-west.service"), true},
		{"/data/namespaces/west/runtime/skupper-west.service", false},
		{filepath.Join("/data/namespaces/west" + scriptsSegment + "other.service"), false},
		{filepath.Join("/data/namespaces/west", string(api.ScriptsPath), "skupper-west.sh"), false},
	}

	for _, tc := range cases {
		got := e.Filter(tc.path)
		assert.Equal(t, tc.want, got, "Filter(%q)", tc.path)
	}
}

func TestSystemdUnitPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	eRoot, _ := newTestEnabler(t, 0)
	assert.Equal(t, filepath.Join("/etc/systemd/system", "skupper-west.service"),
		eRoot.systemdUnitPath("skupper-west.service"))

	eUser, _ := newTestEnabler(t, 1000)
	assert.Equal(t, filepath.Join(dir, "systemd", "user", "skupper-west.service"),
		eUser.systemdUnitPath("skupper-west.service"))
}

func TestSystemctlArgs(t *testing.T) {
	for _, uid := range []int{0, 1000} {
		uid := uid
		t.Run(fmt.Sprintf("uid-%d", uid), func(t *testing.T) {
			e, calls := newTestEnabler(t, uid)
			_ = e.systemctl("enable", "skupper-west.service")
			assert.Assert(t, len(*calls) == 1)
			args := (*calls)[0]
			assert.Equal(t, "systemctl", args[0])
			hasUser := args[1] == "--user"
			assert.Equal(t, uid != 0, hasUser)
		})
	}
}

func TestEnableService_CopiesAndEnables(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	scriptsDir := filepath.Join(t.TempDir(), "namespaces", "west", string(api.ScriptsPath))
	assert.Assert(t, os.MkdirAll(scriptsDir, 0755))
	srcPath := filepath.Join(scriptsDir, "skupper-west.service")
	assert.Assert(t, os.WriteFile(srcPath, []byte("[Unit]\nDescription=test\n"), 0644))

	e, calls := newTestEnabler(t, 1000)
	e.enableService(srcPath)

	dstPath := filepath.Join(configDir, "systemd", "user", "skupper-west.service")
	data, err := os.ReadFile(dstPath)
	assert.Assert(t, err)
	assert.Equal(t, "[Unit]\nDescription=test\n", string(data))

	assert.Assert(t, len(*calls) == 2, "expected 2 systemctl calls, got %d", len(*calls))
	assert.Assert(t, strings.Contains(strings.Join((*calls)[0], " "), "daemon-reload"))
	assert.Assert(t, strings.Contains(strings.Join((*calls)[1], " "), "enable"))
	assert.Assert(t, strings.Contains(strings.Join((*calls)[1], " "), "--now"))
	assert.Assert(t, strings.Contains(strings.Join((*calls)[1], " "), "skupper-west.service"))
}

func TestEnableService_SkipsWriteWhenUnchanged(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	content := []byte("[Unit]\nDescription=test\n")
	scriptsDir := filepath.Join(t.TempDir(), "namespaces", "west", string(api.ScriptsPath))
	assert.Assert(t, os.MkdirAll(scriptsDir, 0755))
	srcPath := filepath.Join(scriptsDir, "skupper-west.service")
	assert.Assert(t, os.WriteFile(srcPath, content, 0644))

	dstDir := filepath.Join(configDir, "systemd", "user")
	assert.Assert(t, os.MkdirAll(dstDir, 0755))
	dstPath := filepath.Join(dstDir, "skupper-west.service")
	assert.Assert(t, os.WriteFile(dstPath, content, 0644))
	info, err := os.Stat(dstPath)
	assert.Assert(t, err)
	modBefore := info.ModTime()

	e, _ := newTestEnabler(t, 1000)
	e.enableService(srcPath)

	info, err = os.Stat(dstPath)
	assert.Assert(t, err)
	assert.Equal(t, modBefore, info.ModTime(), "file should not have been rewritten")
}

func TestEnableService_MissingSource(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	e, calls := newTestEnabler(t, 1000)
	e.enableService("/nonexistent/scripts/skupper-west.service")

	assert.Equal(t, 0, len(*calls), "expected no systemctl calls for missing source")
}

func TestOnCreate_CallsEnableService(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	scriptsDir := filepath.Join(t.TempDir(), "namespaces", "east", string(api.ScriptsPath))
	assert.Assert(t, os.MkdirAll(scriptsDir, 0755))
	srcPath := filepath.Join(scriptsDir, "skupper-east.service")
	assert.Assert(t, os.WriteFile(srcPath, []byte("[Unit]\n"), 0644))

	e, calls := newTestEnabler(t, 1000)
	e.OnCreate(srcPath)

	dstPath := filepath.Join(configDir, "systemd", "user", "skupper-east.service")
	_, err := os.ReadFile(dstPath)
	assert.Assert(t, err)
	assert.Assert(t, len(*calls) >= 1)
}

func TestOnUpdate_CallsEnableService(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	scriptsDir := filepath.Join(t.TempDir(), "namespaces", "east", string(api.ScriptsPath))
	assert.Assert(t, os.MkdirAll(scriptsDir, 0755))
	srcPath := filepath.Join(scriptsDir, "skupper-east.service")
	assert.Assert(t, os.WriteFile(srcPath, []byte("[Unit]\n"), 0644))

	e, calls := newTestEnabler(t, 1000)
	e.OnUpdate(srcPath)

	dstPath := filepath.Join(configDir, "systemd", "user", "skupper-east.service")
	_, err := os.ReadFile(dstPath)
	assert.Assert(t, err)
	assert.Assert(t, len(*calls) >= 1)
}

func TestOnRemove_DisablesAndDeletesUnit(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	unitDir := filepath.Join(configDir, "systemd", "user")
	assert.Assert(t, os.MkdirAll(unitDir, 0755))
	dstPath := filepath.Join(unitDir, "skupper-west.service")
	assert.Assert(t, os.WriteFile(dstPath, []byte("[Unit]\n"), 0644))

	e, calls := newTestEnabler(t, 1000)
	e.OnRemove(dstPath)

	_, err := os.Stat(dstPath)
	assert.Assert(t, os.IsNotExist(err), "expected unit file to be removed")

	joined := make([]string, len(*calls))
	for i, c := range *calls {
		joined[i] = strings.Join(c, " ")
	}
	all := strings.Join(joined, " | ")
	assert.Assert(t, strings.Contains(all, "disable"), "expected disable call, got: %s", all)
	assert.Assert(t, strings.Contains(all, "daemon-reload"), "expected daemon-reload call, got: %s", all)
}

func TestNamespacesHandlerFilter(t *testing.T) {
	namespacesDir := t.TempDir()

	subDir := filepath.Join(namespacesDir, "west")
	assert.Assert(t, os.MkdirAll(subDir, 0755))
	nestedDir := filepath.Join(subDir, "nested")
	assert.Assert(t, os.MkdirAll(nestedDir, 0755))
	filePath := filepath.Join(namespacesDir, "somefile")
	assert.Assert(t, os.WriteFile(filePath, []byte{}, 0644))

	h := &NamespaceScriptHandler{NamespacesDir: namespacesDir}

	assert.Equal(t, true, h.Filter(subDir), "direct subdir should pass")
	assert.Equal(t, false, h.Filter(nestedDir), "nested dir should be rejected")
	assert.Equal(t, false, h.Filter(filePath), "file should be rejected")
	assert.Equal(t, false, h.Filter(namespacesDir), "the namespaces dir itself should be rejected")
}

func newTestEnabler(t *testing.T, uid int) (*ServiceEnabler, *[][]string) {
	t.Helper()
	var mu sync.Mutex
	var calls [][]string
	e := &ServiceEnabler{
		uid: uid,
		command: func(name string, args ...string) *exec.Cmd {
			mu.Lock()
			calls = append(calls, append([]string{name}, args...))
			mu.Unlock()
			return exec.Command("true")
		},
	}
	return e, &calls
}
