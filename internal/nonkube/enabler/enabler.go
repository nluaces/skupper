package enabler

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/skupperproject/skupper/internal/filesystem"
	"github.com/skupperproject/skupper/pkg/nonkube/api"
)

const serviceFilePattern = "skupper-*.service"

type ServiceEnabler struct {
	uid     int
	command func(string, ...string) *exec.Cmd
	mu      sync.Mutex
}

func NewServiceEnabler() *ServiceEnabler {
	return &ServiceEnabler{
		uid:     os.Getuid(),
		command: exec.Command,
	}
}

func (e *ServiceEnabler) OnCreate(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enableService(name)
}

func (e *ServiceEnabler) OnUpdate(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enableService(name)
}

func (e *ServiceEnabler) OnRemove(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	svcName := filepath.Base(name)
	dst := e.systemdUnitPath(svcName)
	_ = e.systemctl("disable", "--now", svcName)
	_ = os.Remove(dst)
	_ = e.systemctl("daemon-reload")
}

func (e *ServiceEnabler) OnBasePathAdded(_ string) {}

func (e *ServiceEnabler) Filter(name string) bool {
	base := filepath.Base(name)
	matched, _ := filepath.Match(serviceFilePattern, base)
	return matched && filepath.Base(filepath.Dir(name)) == filepath.Base(string(api.ScriptsPath))
}

func (e *ServiceEnabler) enableService(srcPath string) {
	svcName := filepath.Base(srcPath)
	unitDir := filepath.Dir(e.systemdUnitPath(svcName))
	dst := filepath.Join(unitDir, svcName)

	src, err := os.ReadFile(srcPath)
	if err != nil {
		slog.Error("failed to read service file", slog.String("path", srcPath), slog.Any("error", err))
		return
	}

	if err = os.MkdirAll(unitDir, 0755); err != nil {
		slog.Error("failed to create systemd unit directory", slog.String("dir", unitDir), slog.Any("error", err))
		return
	}

	existing, readErr := os.ReadFile(dst)
	if readErr != nil || string(existing) != string(src) {
		if err = os.WriteFile(dst, src, 0644); err != nil {
			slog.Error("failed to write service file", slog.String("dst", dst), slog.Any("error", err))
			return
		}
	}

	if err = e.systemctl("daemon-reload"); err != nil {
		slog.Error("failed to reload systemd daemon", slog.Any("error", err))
		return
	}

	if err = e.systemctl("enable", "--now", svcName); err != nil {
		slog.Error("failed to enable service", slog.String("name", svcName), slog.Any("error", err))
	}
}

func (e *ServiceEnabler) systemdUnitPath(name string) string {
	if e.uid == 0 {
		return filepath.Join("/etc/systemd/system", name)
	}
	return filepath.Join(api.GetConfigHome(), "systemd", "user", name)
}

func (e *ServiceEnabler) systemctl(args ...string) error {
	var fullArgs []string
	if e.uid != 0 {
		fullArgs = append(fullArgs, "--user")
	}
	fullArgs = append(fullArgs, args...)
	return e.command("systemctl", fullArgs...).Run()
}

type NamespaceScriptHandler struct {
	NamespacesDir string
	Watcher       *filesystem.FileWatcher
	Enabler       *ServiceEnabler
}

func (n *NamespaceScriptHandler) OnCreate(name string) {
	scriptsPath := filepath.Join(name, string(api.ScriptsPath))
	n.Watcher.Add(scriptsPath, n.Enabler)
}

func (n *NamespaceScriptHandler) OnUpdate(name string) {
	scriptsPath := filepath.Join(name, string(api.ScriptsPath))
	n.Watcher.Add(scriptsPath, n.Enabler)
}
func (n *NamespaceScriptHandler) OnRemove(_ string)        {}
func (n *NamespaceScriptHandler) OnBasePathAdded(_ string) {}

func (n *NamespaceScriptHandler) Filter(name string) bool {
	if filepath.Dir(name) != n.NamespacesDir {
		return false
	}
	stat, err := os.Stat(name)
	if err != nil {
		return false
	}
	return stat.IsDir()
}
