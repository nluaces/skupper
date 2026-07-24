package networkobserver

import "fmt"

func RenderPrometheusConfig(netobsPort int) string {
	return fmt.Sprintf(`global:
  scrape_interval: 15s
  evaluation_interval: 15s
alerting:
  alertmanagers:
    - static_configs:
        - targets:
scrape_configs:
  - job_name: "network-observer-local"
    scheme: http
    follow_redirects: true
    enable_http2: true
    static_configs:
      - targets: ["localhost:%d"]
`, netobsPort)
}
