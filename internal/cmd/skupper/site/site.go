/*
Copyright © 2024 Skupper Team <skupper@googlegroups.com>
*/
package site

import (
	"github.com/skupperproject/skupper/api/types"
	"github.com/skupperproject/skupper/internal/cmd/skupper/common"
	"github.com/skupperproject/skupper/internal/cmd/skupper/site/kube"
	"github.com/skupperproject/skupper/internal/cmd/skupper/site/non_kube"
	"github.com/skupperproject/skupper/pkg/config"
	"github.com/spf13/cobra"
)

func NewCmdSite() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "site",
		Short: "A site is where skupper is deployed and components of your application are running.",
		Long:  `A site is a place where components of your application are running. Sites are linked to form application networks.`,
		Example: `skupper site create my-site
skupper site status`,
	}

	cmd.AddCommand(CmdSiteCreateFactory(config.GetPlatform()))
	cmd.AddCommand(CmdSiteUpdateFactory(config.GetPlatform()))
	cmd.AddCommand(CmdSiteStatusFactory(config.GetPlatform()))
	cmd.AddCommand(CmdSiteDeleteFactory(config.GetPlatform()))

	return cmd
}

func CmdSiteCreateFactory(configuredPlatform types.Platform) *cobra.Command {
	kubeCommand := kube.NewCmdSiteCreate()
	nonKubeCommand := non_kube.NewCmdSiteCreate()

	cmdSiteCreateDesc := common.SkupperCmdDescription{
		Use:   "create <name>",
		Short: "Create a new site",
		Long: `A site is a place where components of your application are running.
Sites are linked to form application networks.
There can be only one site definition per namespace.`,
	}

	cmd := common.ConfigureCobraCommand(configuredPlatform, cmdSiteCreateDesc, kubeCommand, nonKubeCommand)

	cmdFlags := common.CommandSiteCreateFlags{}

	cmd.Flags().BoolVar(&cmdFlags.EnableLinkAccess, common.FlagNameEnableLinkAccess, false, common.FlagDescEnableLinkAccess)
	cmd.Flags().StringVar(&cmdFlags.LinkAccessType, common.FlagNameLinkAccessType, "", common.FlagDescLinkAccessType)
	cmd.Flags().StringVarP(&cmdFlags.Output, common.FlagNameOutput, "o", "", common.FlagDescOutput)
	cmd.Flags().StringVar(&cmdFlags.ServiceAccount, common.FlagNameServiceAccount, "", common.FlagDescServiceAccount)

	kubeCommand.CobraCmd = cmd
	kubeCommand.Flags = &cmdFlags
	nonKubeCommand.CobraCmd = cmd
	nonKubeCommand.Flags = &cmdFlags

	return cmd

}

func CmdSiteUpdateFactory(configuredPlatform types.Platform) *cobra.Command {
	kubeCommand := kube.NewCmdSiteUpdate()
	nonKubeCommand := non_kube.NewCmdSiteUpdate()

	cmdSiteUpdateDesc := common.SkupperCmdDescription{
		Use:     "update <name>",
		Short:   "Change site settings",
		Long:    `Change site settings of a given site.`,
		Example: "skupper site update my-site --enable-link-access",
	}

	cmd := common.ConfigureCobraCommand(configuredPlatform, cmdSiteUpdateDesc, kubeCommand, nonKubeCommand)
	cmdFlags := common.CommandSiteUpdateFlags{}

	cmd.Flags().BoolVar(&cmdFlags.EnableLinkAccess, common.FlagNameEnableLinkAccess, false, common.FlagDescEnableLinkAccess)
	cmd.Flags().StringVar(&cmdFlags.LinkAccessType, common.FlagNameLinkAccessType, "", common.FlagDescLinkAccessType)
	cmd.Flags().StringVarP(&cmdFlags.Output, common.FlagNameOutput, "o", "", common.FlagDescOutput)
	cmd.Flags().StringVar(&cmdFlags.ServiceAccount, common.FlagNameServiceAccount, "", common.FlagDescServiceAccount)

	kubeCommand.CobraCmd = cmd
	kubeCommand.Flags = &cmdFlags
	nonKubeCommand.CobraCmd = cmd
	nonKubeCommand.Flags = &cmdFlags

	return cmd
}

func CmdSiteStatusFactory(configuredPlatform types.Platform) *cobra.Command {
	kubeCommand := kube.NewCmdSiteStatus()
	nonKubeCommand := non_kube.NewCmdSiteStatus()

	cmdSiteStatusDesc := common.SkupperCmdDescription{
		Use:   "status",
		Short: "Get the site status",
		Long:  `Display the current status of a site.`,
	}

	cmd := common.ConfigureCobraCommand(configuredPlatform, cmdSiteStatusDesc, kubeCommand, nonKubeCommand)

	kubeCommand.CobraCmd = cmd
	nonKubeCommand.CobraCmd = cmd

	return cmd

}

func CmdSiteDeleteFactory(configuredPlatform types.Platform) *cobra.Command {
	kubeCommand := kube.NewCmdSiteDelete()
	nonKubeCommand := non_kube.NewCmdSiteDelete()

	cmdSiteDeleteDesc := common.SkupperCmdDescription{
		Use:     "delete",
		Short:   "Delete a site",
		Long:    `Delete a site by name`,
		Example: "skupper site delete my-site",
	}

	cmd := common.ConfigureCobraCommand(configuredPlatform, cmdSiteDeleteDesc, kubeCommand, nonKubeCommand)

	kubeCommand.CobraCmd = cmd
	nonKubeCommand.CobraCmd = cmd

	return cmd
}