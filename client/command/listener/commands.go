package listener

import (
	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *core.Console) []*cobra.Command {
	listenerCmd := &cobra.Command{
		Use:   consts.CommandListener,
		Short: "List listeners on the server",
		Long:  "List listeners on the server in table form.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListenerCmd(cmd, con)
		},
		Annotations: map[string]string{
			"resource": "true",
		},
		Example: `~~~
listener
~~~`,
	}

	jobCmd := &cobra.Command{
		Use:   consts.CommandJob,
		Short: "List jobs on the server",
		Long:  "List jobs on the server in table form.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListJobsCmd(cmd, con)
		},
		Annotations: map[string]string{
			"resource": "true",
		},
		Example: `~~~
job
~~~`,
	}
	jobInspectCmd := &cobra.Command{
		Use:   "inspect [job]",
		Short: "Inspect a running job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return InspectJobCmd(cmd, con)
		},
	}
	jobKillCmd := &cobra.Command{
		Use:   "kill [job]",
		Short: "Stop a running job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return KillJobCmd(cmd, con)
		},
	}
	common.BindArgCompletions(jobInspectCmd, nil, common.AllPipelineCompleter(con))
	common.BindArgCompletions(jobKillCmd, nil, common.AllPipelineCompleter(con))
	jobCmd.AddCommand(jobInspectCmd, jobKillCmd)

	pipelineCmd := &cobra.Command{
		Use:   consts.CommandPipeline,
		Short: "Manage pipelines",
		Long:  "Start, stop, list, and delete server pipelines.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	startPipelineCmd := &cobra.Command{
		Use:   consts.CommandPipelineStart,
		Short: "Start a pipeline",
		Args:  cobra.ExactArgs(1),
		Long:  "Start the specified pipeline.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StartPipelineCmd(cmd, con)
		},
		Example: `~~~
pipeline start tcp_test
~~~`,
	}

	common.BindArgCompletions(startPipelineCmd, nil, common.AllPipelineCompleter(con))
	common.BindFlag(startPipelineCmd, func(f *pflag.FlagSet) {
		f.String("cert-name", "", "certificate name")
	})
	common.BindFlagCompletions(startPipelineCmd, func(comp carapace.ActionMap) {
		comp["cert-name"] = common.CertNameCompleter(con)
	})

	stopPipelineCmd := &cobra.Command{
		Use:   consts.CommandPipelineStop,
		Short: "Stop a pipeline",
		Args:  cobra.ExactArgs(1),
		Long:  "Stop the specified pipeline.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StopPipelineCmd(cmd, con)
		},
		Example: `~~~
pipeline stop tcp_test
~~~`,
	}

	common.BindArgCompletions(stopPipelineCmd, nil, common.AllPipelineCompleter(con))

	listPipelineCmd := &cobra.Command{
		Use:   consts.CommandPipelineList,
		Short: "List pipelines",
		Long:  "List pipelines for all listeners or for a specific listener.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListPipelineCmd(cmd, con)
		},
		Example: `
list all pipelines
~~~
pipeline list
~~~

list pipelines in listener
~~~
pipeline list listener_id
~~~`,
	}

	deletePipeCmd := &cobra.Command{
		Use:   consts.CommandPipelineDelete + " [pipeline]",
		Short: "Delete a pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return DeletePipelineCmd(cmd, con)
		},
	}

	common.BindArgCompletions(deletePipeCmd, nil, common.AllPipelineCompleter(con))

	inspectPipelineCmd := &cobra.Command{
		Use:   "inspect [pipeline]",
		Short: "Inspect a pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return InspectPipelineCmd(cmd, con)
		},
	}
	common.BindArgCompletions(inspectPipelineCmd, nil, common.AllPipelineCompleter(con))

	restartPipelineCmd := &cobra.Command{
		Use:   "restart [pipeline]",
		Short: "Restart a pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RestartPipelineCmd(cmd, con)
		},
	}
	common.BindArgCompletions(restartPipelineCmd, nil, common.AllPipelineCompleter(con))

	updatePipelineCmd := &cobra.Command{
		Use:   "update [pipeline]",
		Short: "Update cached pipeline metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdatePipelineCmd(cmd, con)
		},
	}
	common.BindFlag(updatePipelineCmd, func(f *pflag.FlagSet) {
		f.Bool("enable", false, "enable pipeline")
		f.Bool("disable", false, "disable pipeline")
		f.String("cert-name", "", "certificate name")
		f.String("parser", "", "pipeline parser")
	})
	common.BindArgCompletions(updatePipelineCmd, nil, common.AllPipelineCompleter(con))
	common.BindFlagCompletions(updatePipelineCmd, func(comp carapace.ActionMap) {
		comp["cert-name"] = common.CertNameCompleter(con)
	})

	healthPipelineCmd := &cobra.Command{
		Use:   "health",
		Short: "Show pipeline health summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			return HealthPipelineCmd(cmd, con)
		},
	}
	common.BindFlag(healthPipelineCmd, func(f *pflag.FlagSet) {
		f.String("listener", "", "listener ID")
	})
	common.BindFlagCompletions(healthPipelineCmd, func(comp carapace.ActionMap) {
		comp["listener"] = common.ListenerIDCompleter(con)
	})

	pipelineCmd.AddCommand(startPipelineCmd, stopPipelineCmd, listPipelineCmd, deletePipeCmd,
		inspectPipelineCmd, restartPipelineCmd, updatePipelineCmd, healthPipelineCmd)

	forwardCmd := &cobra.Command{
		Use:   "forward",
		Short: "Manage forward listeners",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	forwardConnectCmd := &cobra.Command{
		Use:   "connect [listener_id]",
		Short: "Connect to a forward listener",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ForwardConnectCmd(cmd, con)
		},
		Example: `~~~
listener forward connect listener --host 10.0.0.5 --port 5005
~~~`,
	}
	common.BindFlag(forwardConnectCmd, func(f *pflag.FlagSet) {
		f.String("host", "", "forward listener host")
		f.Uint16("port", 5005, "forward listener port")
		f.Uint32("timeout", 5, "connect timeout in seconds")
	})
	_ = forwardConnectCmd.MarkFlagRequired("host")
	common.BindArgCompletions(forwardConnectCmd, nil, common.ListenerIDCompleter(con))
	common.BindFlagCompletions(forwardConnectCmd, func(comp carapace.ActionMap) {
		comp["host"] = carapace.ActionValues().Usage("forward listener host")
		comp["port"] = carapace.ActionValues("5005").Usage("forward listener port")
		comp["timeout"] = carapace.ActionValues("5", "10", "30").Usage("connect timeout in seconds")
	})

	forwardDisconnectCmd := &cobra.Command{
		Use:   "disconnect [listener_id]",
		Short: "Disconnect a forward listener",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ForwardDisconnectCmd(cmd, con)
		},
	}
	common.BindArgCompletions(forwardDisconnectCmd, nil, common.ForwardListenerIDCompleter(con))

	forwardStatusCmd := &cobra.Command{
		Use:   "status [listener_id]",
		Short: "Show forward listener status",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ForwardStatusCmd(cmd, con)
		},
	}
	common.BindArgCompletions(forwardStatusCmd, nil, common.ForwardListenerIDCompleter(con))

	forwardListCmd := &cobra.Command{
		Use:   "list",
		Short: "List connected forward listeners",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ForwardListCmd(cmd, con)
		},
	}
	forwardCmd.AddCommand(forwardConnectCmd, forwardDisconnectCmd, forwardStatusCmd, forwardListCmd)

	retireCmd := &cobra.Command{
		Use:   "retire [listener_id]",
		Short: "Retire a listener",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RetireListenerCmd(cmd, con)
		},
		Example: `~~~
listener retire listener-a --purge-config --purge-auth --yes
~~~`,
	}
	common.BindFlag(retireCmd, func(f *pflag.FlagSet) {
		f.Bool("purge-config", false, "remove the listener config file before shutdown")
		f.Bool("purge-auth", false, "remove the listener auth file before shutdown")
		f.Bool("no-revoke", false, "do not revoke the listener operator after retirement")
		f.Uint32("timeout", 10, "retire timeout in seconds")
	})
	common.BindArgCompletions(retireCmd, nil, common.ListenerIDCompleter(con))
	common.BindFlagCompletions(retireCmd, func(comp carapace.ActionMap) {
		comp["timeout"] = carapace.ActionValues("5", "10", "30").Usage("retire timeout in seconds")
	})

	inspectListenerCmd := &cobra.Command{
		Use:   "inspect [listener_id]",
		Short: "Inspect a listener",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return InspectListenerCmd(cmd, con)
		},
	}
	common.BindArgCompletions(inspectListenerCmd, nil, common.ListenerIDCompleter(con))

	listenerCmd.AddCommand(forwardCmd, retireCmd, inspectListenerCmd)

	return []*cobra.Command{listenerCmd, jobCmd, pipelineCmd}
}
