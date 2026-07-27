package networkobserver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const SystemdServiceTemplate = `[Unit]
Description=Skupper Network Observer - %s
After=network.target

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/bin/true
ExecStop=/bin/true

[Install]
WantedBy=default.target
`

const SystemdPrometheusServiceTemplate = `[Unit]
Description=Skupper Network Observer Prometheus - %s
After=network.target
PartOf=skupper-network-observer-%s.service

[Service]
Type=simple
Restart=always
RestartSec=5
ExecStart=%s start --attach %s-skupper-prometheus
ExecStop=%s stop %s-skupper-prometheus

[Install]
WantedBy=skupper-network-observer-%s.service
`

const SystemdNetworkObserverServiceTemplate = `[Unit]
Description=Skupper Network Observer Application - %s
After=network.target skupper-controller.service skupper-network-observer-prometheus-%s.service
Wants=skupper-controller.service
PartOf=skupper-network-observer-%s.service

[Service]
Type=simple
Restart=always
RestartSec=5
ExecStart=%s start --attach %s-skupper-network-observer
ExecStop=%s stop %s-skupper-network-observer

[Install]
WantedBy=skupper-network-observer-%s.service
`

type SystemdServiceManager struct {
	Namespace       string
	ContainerEngine string
	ServiceDir      string
}

func NewSystemdServiceManager(namespace, containerEngine string, _ ports) *SystemdServiceManager {
	serviceDir := getSystemdServiceDir()
	return &SystemdServiceManager{
		Namespace:       namespace,
		ContainerEngine: containerEngine,
		ServiceDir:      serviceDir,
	}
}

func getSystemdServiceDir() string {
	if os.Getuid() == 0 {
		return "/etc/systemd/system"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = fmt.Sprintf("/home/%s", os.Getenv("USER"))
	}
	return filepath.Join(home, ".config", "systemd", "user")
}

func (s *SystemdServiceManager) CreateServices() error {

	if err := os.MkdirAll(s.ServiceDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd service directory: %w", err)
	}

	mainServiceName := fmt.Sprintf("skupper-network-observer-%s.service", s.Namespace)
	mainServicePath := filepath.Join(s.ServiceDir, mainServiceName)
	mainServiceContent := fmt.Sprintf(SystemdServiceTemplate, s.Namespace)
	if err := os.WriteFile(mainServicePath, []byte(mainServiceContent), 0644); err != nil {
		return fmt.Errorf("failed to write main service file: %w", err)
	}

	prometheusServiceName := fmt.Sprintf("skupper-network-observer-prometheus-%s.service", s.Namespace)
	prometheusServicePath := filepath.Join(s.ServiceDir, prometheusServiceName)
	prometheusServiceContent := fmt.Sprintf(SystemdPrometheusServiceTemplate,
		s.Namespace, s.Namespace,
		s.ContainerEngine, s.Namespace,
		s.ContainerEngine, s.Namespace,
		s.Namespace)
	if err := os.WriteFile(prometheusServicePath, []byte(prometheusServiceContent), 0644); err != nil {
		return fmt.Errorf("failed to write prometheus service file: %w", err)
	}

	appServiceName := fmt.Sprintf("skupper-network-observer-app-%s.service", s.Namespace)
	appServicePath := filepath.Join(s.ServiceDir, appServiceName)
	appServiceContent := fmt.Sprintf(SystemdNetworkObserverServiceTemplate,
		s.Namespace, s.Namespace, s.Namespace,
		s.ContainerEngine, s.Namespace,
		s.ContainerEngine, s.Namespace,
		s.Namespace)
	if err := os.WriteFile(appServicePath, []byte(appServiceContent), 0644); err != nil {
		return fmt.Errorf("failed to write network observer service file: %w", err)
	}

	for _, svc := range []string{
		prometheusServiceName,
		appServiceName,
		mainServiceName,
	} {
		if err := s.enableService(svc); err != nil {
			return fmt.Errorf("failed to enable service %s: %w", svc, err)
		}
	}

	for _, svc := range []string{
		prometheusServiceName,
		appServiceName,
	} {
		if err := s.startService(svc); err != nil {
			return fmt.Errorf("failed to start service %s: %w", svc, err)
		}
	}

	if err := s.startService(mainServiceName); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

func (s *SystemdServiceManager) reloadSystemd() error {
	var cmd *exec.Cmd
	if os.Getuid() == 0 {
		cmd = exec.Command("systemctl", "daemon-reload")
	} else {
		cmd = exec.Command("systemctl", "--user", "daemon-reload")
	}
	return cmd.Run()
}

func (s *SystemdServiceManager) enableService(serviceName string) error {
	var cmd *exec.Cmd
	if os.Getuid() == 0 {
		cmd = exec.Command("systemctl", "enable", serviceName)
	} else {
		cmd = exec.Command("systemctl", "--user", "enable", serviceName)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable %s: %w", serviceName, err)
	}
	return nil
}

func (s *SystemdServiceManager) startService(serviceName string) error {
	var cmd *exec.Cmd
	if os.Getuid() == 0 {
		cmd = exec.Command("systemctl", "start", serviceName)
	} else {
		cmd = exec.Command("systemctl", "--user", "start", serviceName)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start %s: %w", serviceName, err)
	}
	return nil
}

func (s *SystemdServiceManager) RemoveServices() error {
	mainServiceName := fmt.Sprintf("skupper-network-observer-%s.service", s.Namespace)


	if err := s.stopAndDisableService(mainServiceName); err != nil {
		fmt.Printf("Warning: failed to stop service: %v\n", err)
	}

	
	serviceNames := []string{
		mainServiceName,
		fmt.Sprintf("skupper-network-observer-prometheus-%s.service", s.Namespace),
		fmt.Sprintf("skupper-network-observer-app-%s.service", s.Namespace),
	}

	for _, serviceName := range serviceNames {
		servicePath := filepath.Join(s.ServiceDir, serviceName)
		if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove service file %s: %w", serviceName, err)
		}
	}

	if err := s.reloadSystemd(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	return nil
}

func (s *SystemdServiceManager) stopAndDisableService(serviceName string) error {
	var stopCmd, disableCmd *exec.Cmd
	if os.Getuid() == 0 {
		stopCmd = exec.Command("systemctl", "stop", serviceName)
		disableCmd = exec.Command("systemctl", "disable", serviceName)
	} else {
		stopCmd = exec.Command("systemctl", "--user", "stop", serviceName)
		disableCmd = exec.Command("systemctl", "--user", "disable", serviceName)
	}

	err := stopCmd.Run()
	if err != nil {
		return err
	}

	err = disableCmd.Run()
	if err != nil {
		return err
	}

	return nil
}
