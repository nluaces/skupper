package nonkube

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/skupperproject/skupper/internal/filesystem"
	"github.com/skupperproject/skupper/internal/nonkube/enabler"
	"github.com/skupperproject/skupper/internal/version"
	"github.com/skupperproject/skupper/pkg/nonkube/api"
	"github.com/spf13/cobra"
)

func NewCmdSiteServiceEnabler() *cobra.Command {
	return &cobra.Command{
		Use:    "_site-service-enabler",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Starting site-service-enabler", slog.String("version", version.Version))

			stop := make(chan struct{})

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				<-sigs
				slog.Info("Shutting down site-service-enabler")
				close(stop)
			}()

			return run(stop)
		},
	}
}

func run(stop <-chan struct{}) error {
	namespacesDir := api.GetDefaultOutputNamespacesPath()

	watcher, err := filesystem.NewWatcher(slog.String("component", "site-service-enabler"))
	if err != nil {
		return err
	}

	siteServiceEnabler := enabler.NewServiceEnabler()
	//this checks that the script directory is created, where the systemd services are going to be stored
	nsHandler := &enabler.NamespaceScriptHandler{
		NamespacesDir: namespacesDir,
		Watcher:       watcher,
		Enabler:       siteServiceEnabler,
	}
	watcher.Add(namespacesDir, nsHandler)
	watcher.Start(stop)

	<-stop
	return nil
}
