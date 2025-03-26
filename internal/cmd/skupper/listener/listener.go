package listener

import (
	"time"

	"github.com/skupperproject/skupper/internal/cmd/skupper/common"
	"github.com/skupperproject/skupper/internal/cmd/skupper/listener/kube"
	"github.com/skupperproject/skupper/internal/cmd/skupper/listener/nonkube"
	"github.com/skupperproject/skupper/internal/config"
	"github.com/spf13/cobra"
)

func NewCmdListener() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "listener",
		Short: "Binds a connection endpoint in the local site to target workloads in remote sites.",
		Long:  `A listener is a connection endpoint in the local site and binds to target workloads in remote sites`,
		Example: `skupper listener create my-listener 8080
skupper listener status my-listener`,
	}

	platform := common.Platform(config.GetPlatform())
	cmd.AddCommand(CmdListenerCreateFactory(platform))
	cmd.AddCommand(CmdListenerStatusFactory(platform))
	cmd.AddCommand(CmdListenerUpdateFactory(platform))
	cmd.AddCommand(CmdListenerDeleteFactory(platform))
	cmd.AddCommand(CmdListenerGenerateFactory(platform))

	return cmd
}

func CmdListenerCreateFactory(configuredPlatform common.Platform) *cobra.Command {
	kubeCommand := kube.NewCmdListenerCreate()
	nonKubeCommand := nonkube.NewCmdListenerCreate()

	cmdListenerCreateDesc := common.SkupperCmdDescription{
		Use:     "create <name> <port>",
		Short:   "create a listener",
		Long:    "Clients at this site use the listener host and port to establish connections to the remote service.",
		Example: "skupper listener create database 5432",
	}

	cmd := common.ConfigureCobraCommand(configuredPlatform, cmdListenerCreateDesc, kubeCommand, nonKubeCommand)

	cmdFlags := common.CommandListenerCreateFlags{}

	cmd.Flags().StringVar(&cmdFlags.RoutingKey, common.FlagNameRoutingKey, "", common.FlagDescRoutingKey)
	cmd.Flags().StringVar(&cmdFlags.Host, common.FlagNameListenerHost, "", common.FlagDescListenerHost)
	cmd.Flags().StringVar(&cmdFlags.TlsCredentials, common.FlagNameTlsCredentials, "", common.FlagDescTlsCredentials)
	cmd.Flags().StringVar(&cmdFlags.ListenerType, common.FlagNameListenerType, "tcp", common.FlagDescListenerType)

	if configuredPlatform == common.PlatformKubernetes {
		cmd.Flags().DurationVar(&cmdFlags.Timeout, common.FlagNameTimeout, 60*time.Second, common.FlagDescTimeout)
		cmd.Flags().StringVar(&cmdFlags.Wait, common.FlagNameWait, "configured", common.FlagDescWait)
	}

	kubeCommand.CobraCmd = cmd
	kubeCommand.Flags = &cmdFlags
	nonKubeCommand.CobraCmd = cmd
	nonKubeCommand.Flags = &cmdFlags

	return cmd
}

func CmdListenerUpdateFactory(configuredPlatform common.Platform) *cobra.Command {
	kubeCommand := kube.NewCmdListenerUpdate()
	nonKubeCommand := nonkube.NewCmdListenerUpdate()

	cmdListenerUpdateDesc := common.SkupperCmdDescription{
		Use:   "update <name>",
		Short: "update a listener",
		Long: `Clients at this site use the listener host and port to establish connections to the remote service.
	The user can change port, host name, TLS credentials, listener type and routing key`,
		Example: "skupper listener update database --host mysql --port 3306",
	}

	cmd := common.ConfigureCobraCommand(configuredPlatform, cmdListenerUpdateDesc, kubeCommand, nonKubeCommand)

	cmdFlags := common.CommandListenerUpdateFlags{}

	cmd.Flags().StringVar(&cmdFlags.RoutingKey, common.FlagNameRoutingKey, "", common.FlagDescRoutingKey)
	cmd.Flags().StringVar(&cmdFlags.Host, common.FlagNameListenerHost, "", common.FlagDescListenerHost)
	cmd.Flags().StringVar(&cmdFlags.TlsCredentials, common.FlagNameTlsCredentials, "", common.FlagDescTlsCredentials)
	cmd.Flags().StringVar(&cmdFlags.ListenerType, common.FlagNameListenerType, "tcp", common.FlagDescListenerType)
	cmd.Flags().IntVar(&cmdFlags.Port, common.FlagNameListenerPort, 0, common.FlagDescListenerPort)
	if configuredPlatform == common.PlatformKubernetes {
		cmd.Flags().DurationVar(&cmdFlags.Timeout, common.FlagNameTimeout, 60*time.Second, common.FlagDescTimeout)
		cmd.Flags().StringVar(&cmdFlags.Wait, common.FlagNameWait, "configured", common.FlagDescWait)
	}

	kubeCommand.CobraCmd = cmd
	kubeCommand.Flags = &cmdFlags
	nonKubeCommand.CobraCmd = cmd
	nonKubeCommand.Flags = &cmdFlags

	return cmd
}

func CmdListenerStatusFactory(configuredPlatform common.Platform) *cobra.Command {
	kubeCommand := kube.NewCmdListenerStatus()
	nonKubeCommand := nonkube.NewCmdListenerStatus()

	cmdListenerStatusDesc := common.SkupperCmdDescription{
		Use:     "status <name>",
		Short:   "get status of listeners",
		Long:    "Display status of all listeners or a specific listener",
		Example: "skupper listener status backend",
	}

	cmd := common.ConfigureCobraCommand(configuredPlatform, cmdListenerStatusDesc, kubeCommand, nonKubeCommand)

	cmdFlags := common.CommandListenerStatusFlags{}

	cmd.Flags().StringVarP(&cmdFlags.Output, common.FlagNameOutput, "o", "", common.FlagDescOutput)

	kubeCommand.CobraCmd = cmd
	kubeCommand.Flags = &cmdFlags
	nonKubeCommand.CobraCmd = cmd
	nonKubeCommand.Flags = &cmdFlags

	return cmd
}

func CmdListenerDeleteFactory(configuredPlatform common.Platform) *cobra.Command {
	kubeCommand := kube.NewCmdListenerDelete()
	nonKubeCommand := nonkube.NewCmdListenerDelete()

	cmdListenerDeleteDesc := common.SkupperCmdDescription{
		Use:     "delete <name>",
		Short:   "delete a listener",
		Long:    "Delete a listener <name>",
		Example: "skupper listener delete database",
	}

	cmd := common.ConfigureCobraCommand(configuredPlatform, cmdListenerDeleteDesc, kubeCommand, nonKubeCommand)

	cmdFlags := common.CommandListenerDeleteFlags{}

	if configuredPlatform == common.PlatformKubernetes {
		cmd.Flags().DurationVar(&cmdFlags.Timeout, common.FlagNameTimeout, 60*time.Second, common.FlagDescTimeout)
		cmd.Flags().BoolVar(&cmdFlags.Wait, common.FlagNameWait, true, common.FlagDescDeleteWait)
	}

	kubeCommand.CobraCmd = cmd
	kubeCommand.Flags = &cmdFlags
	nonKubeCommand.CobraCmd = cmd
	nonKubeCommand.Flags = &cmdFlags

	return cmd
}

func CmdListenerGenerateFactory(configuredPlatform common.Platform) *cobra.Command {
	kubeCommand := kube.NewCmdListenerGenerate()
	nonKubeCommand := nonkube.NewCmdListenerGenerate()

	cmdListenerGenerateDesc := common.SkupperCmdDescription{
		Use:   "generate <name> <port>",
		Short: "generate a listener resource and output it to a file or screen",
		Long: `Clients at this site use the listener host and port to establish connections to the remote service.
	generate a listener to evaluate what will be created with listener create command`,
		Example: "skupper listener generate database 5432",
	}

	cmd := common.ConfigureCobraCommand(configuredPlatform, cmdListenerGenerateDesc, kubeCommand, nonKubeCommand)

	cmdFlags := common.CommandListenerGenerateFlags{}

	cmd.Flags().StringVar(&cmdFlags.RoutingKey, common.FlagNameRoutingKey, "", common.FlagDescRoutingKey)
	cmd.Flags().StringVar(&cmdFlags.Host, common.FlagNameListenerHost, "", common.FlagDescListenerHost)
	cmd.Flags().StringVar(&cmdFlags.TlsCredentials, common.FlagNameTlsCredentials, "", common.FlagDescTlsCredentials)
	cmd.Flags().StringVar(&cmdFlags.ListenerType, common.FlagNameListenerType, "tcp", common.FlagDescListenerType)
	cmd.Flags().StringVarP(&cmdFlags.Output, common.FlagNameOutput, "o", "yaml", common.FlagDescOutput)

	kubeCommand.CobraCmd = cmd
	kubeCommand.Flags = &cmdFlags
	nonKubeCommand.CobraCmd = cmd
	nonKubeCommand.Flags = &cmdFlags

	return cmd
}
