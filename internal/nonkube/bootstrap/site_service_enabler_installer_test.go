package bootstrap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func fakeCommand(calls *[][]string) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		*calls = append(*calls, append([]string{name}, args...))
		return exec.Command("true")
	}
}

func newTestInstaller(t *testing.T, uid int, calls *[][]string) *SiteServiceEnablerInstaller {
	t.Helper()
	tmp := t.TempDir()
	return &SiteServiceEnablerInstaller{
		uid:                 uid,
		rootSystemdBasePath: filepath.Join(tmp, "etc", "systemd", "system"),
		scriptDir:           filepath.Join(tmp, "bin"),
		command:             fakeCommand(calls),
	}
}

func TestTemplateData_NonRoot(t *testing.T) {
	s := &SiteServiceEnablerInstaller{uid: 1000}
	d := s.templateData("/some/path/script")
	assert.Equal(t, d.ScriptPath, "/some/path/script")
	assert.Equal(t, d.WantedBy, "default.target")
}

func TestTemplateData_Root(t *testing.T) {
	s := &SiteServiceEnablerInstaller{uid: 0}
	d := s.templateData("/some/path/script")
	assert.Equal(t, d.WantedBy, "multi-user.target")
}

func TestUserSystemdDir_Root(t *testing.T) {
	s := &SiteServiceEnablerInstaller{uid: 0, rootSystemdBasePath: "/etc/systemd/system"}
	assert.Equal(t, s.userSystemdDir(), "/etc/systemd/system")
}

func TestUserSystemdDir_NonRoot(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/fake/config")
	s := &SiteServiceEnablerInstaller{uid: 1000}
	got := s.userSystemdDir()
	assert.Assert(t, strings.HasPrefix(got, "/fake/config"), "expected XDG_CONFIG_HOME prefix, got %s", got)
}

func TestUnitPath(t *testing.T) {
	s := &SiteServiceEnablerInstaller{uid: 0, rootSystemdBasePath: "/etc/systemd/system"}
	got := s.unitPath("skupper-site-service-enabler.service")
	assert.Equal(t, got, "/etc/systemd/system/skupper-site-service-enabler.service")
}

func TestSystemctl_NonRoot_AddsUserFlag(t *testing.T) {
	var calls [][]string
	s := &SiteServiceEnablerInstaller{uid: 1000, command: fakeCommand(&calls)}
	err := s.systemctl("start", "foo.service")
	assert.NilError(t, err)
	assert.Equal(t, len(calls), 1)
	assert.DeepEqual(t, calls[0], []string{"systemctl", "--user", "start", "foo.service"})
}

func TestSystemctl_Root_NoUserFlag(t *testing.T) {
	var calls [][]string
	s := &SiteServiceEnablerInstaller{uid: 0, command: fakeCommand(&calls)}
	err := s.systemctl("start", "foo.service")
	assert.NilError(t, err)
	assert.Equal(t, len(calls), 1)
	assert.DeepEqual(t, calls[0], []string{"systemctl", "start", "foo.service"})
}

func TestRenderFile(t *testing.T) {
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "out.service")
	s := &SiteServiceEnablerInstaller{}
	err := s.renderFile(siteServiceEnablerServiceTemplate, siteServiceEnablerData{
		ScriptPath: "/usr/bin/skupper-site-service-enabler",
		WantedBy:   "multi-user.target",
	}, dst, 0644)
	assert.NilError(t, err)

	content, err := os.ReadFile(dst)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(content), "ExecStart=/usr/bin/skupper-site-service-enabler"))
	assert.Assert(t, strings.Contains(string(content), "WantedBy=multi-user.target"))
}

func TestRenderFile_InvalidTemplate(t *testing.T) {
	tmp := t.TempDir()
	s := &SiteServiceEnablerInstaller{}
	err := s.renderFile("{{.Invalid", siteServiceEnablerData{}, filepath.Join(tmp, "out.service"), 0644)
	assert.Assert(t, err != nil)
}

func TestInstall_CreatesWrapperScript(t *testing.T) {
	var calls [][]string
	s := newTestInstaller(t, 1000, &calls)

	err := s.Install()
	assert.NilError(t, err)

	scriptPath := filepath.Join(s.scriptDir, siteServiceEnablerWrapperScript)
	content, err := os.ReadFile(scriptPath)
	assert.NilError(t, err)
	assert.Equal(t, string(content), "#!/bin/sh\nexec skupper system _site-service-enabler\n")

	info, err := os.Stat(scriptPath)
	assert.NilError(t, err)
	assert.Equal(t, info.Mode(), os.FileMode(0755))
}

func TestInstall_CreatesServiceFile(t *testing.T) {
	var calls [][]string
	s := newTestInstaller(t, 0, &calls)
	_ = os.MkdirAll(s.rootSystemdBasePath, 0755)

	err := s.Install()
	assert.NilError(t, err)

	svcPath := s.unitPath(siteServiceEnablerServiceFile)
	content, err := os.ReadFile(svcPath)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(content), "ExecStart="))
	assert.Assert(t, strings.Contains(string(content), "WantedBy=multi-user.target"))
}

func TestInstall_SystemctlCallOrder(t *testing.T) {
	var calls [][]string
	s := newTestInstaller(t, 0, &calls)
	_ = os.MkdirAll(s.rootSystemdBasePath, 0755)

	err := s.Install()
	assert.NilError(t, err)

	assert.Equal(t, len(calls), 3)
	assert.DeepEqual(t, calls[0], []string{"systemctl", "daemon-reload"})
	assert.DeepEqual(t, calls[1], []string{"systemctl", "enable", siteServiceEnablerServiceFile})
	assert.DeepEqual(t, calls[2], []string{"systemctl", "start", siteServiceEnablerServiceFile})
}

func TestInstall_NonRoot_SystemctlUsesUserFlag(t *testing.T) {
	var calls [][]string
	s := newTestInstaller(t, 1000, &calls)

	err := s.Install()
	assert.NilError(t, err)

	for _, c := range calls {
		assert.Equal(t, c[1], "--user", "expected --user flag in call %v", c)
	}
}

func TestInstall_ScriptDirCreationFailure(t *testing.T) {
	var calls [][]string
	s := newTestInstaller(t, 1000, &calls)
	blocker := s.scriptDir
	_ = os.MkdirAll(filepath.Dir(blocker), 0755)
	_ = os.WriteFile(blocker, []byte("block"), 0644)
	s.scriptDir = filepath.Join(blocker, "subdir")

	err := s.Install()
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "unable to create script directory"))
}

func TestRemove_SystemctlCallOrder(t *testing.T) {
	var calls [][]string
	s := newTestInstaller(t, 0, &calls)

	s.Remove()

	assert.Equal(t, len(calls), 3)
	assert.DeepEqual(t, calls[0], []string{"systemctl", "stop", siteServiceEnablerServiceFile})
	assert.DeepEqual(t, calls[1], []string{"systemctl", "disable", siteServiceEnablerServiceFile})
	assert.DeepEqual(t, calls[2], []string{"systemctl", "daemon-reload"})
}

func TestRemove_DeletesFiles(t *testing.T) {
	var calls [][]string
	s := newTestInstaller(t, 0, &calls)

	svcPath := s.unitPath(siteServiceEnablerServiceFile)
	_ = os.MkdirAll(filepath.Dir(svcPath), 0755)
	_ = os.WriteFile(svcPath, []byte("[Unit]"), 0644)

	scriptPath := filepath.Join(s.scriptDir, siteServiceEnablerWrapperScript)
	_ = os.MkdirAll(s.scriptDir, 0755)
	_ = os.WriteFile(scriptPath, []byte("#!/bin/sh"), 0755)

	s.Remove()

	_, err := os.Stat(svcPath)
	assert.Assert(t, os.IsNotExist(err), "service file should be removed")

	_, err = os.Stat(scriptPath)
	assert.Assert(t, os.IsNotExist(err), "wrapper script should be removed")
}

func TestRemove_NonRoot_SystemctlUsesUserFlag(t *testing.T) {
	var calls [][]string
	s := newTestInstaller(t, 1000, &calls)

	s.Remove()

	for _, c := range calls {
		assert.Equal(t, c[1], "--user", "expected --user flag in call %v", c)
	}
}

func TestRemove_ToleratesMissingFiles(t *testing.T) {
	var calls [][]string
	s := newTestInstaller(t, 0, &calls)
	s.Remove()
}
