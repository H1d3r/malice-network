package listener

import (
	"fmt"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func RetireListenerCmd(cmd *cobra.Command, con *core.Console) error {
	purgeConfig, err := cmd.Flags().GetBool("purge-config")
	if err != nil {
		return err
	}
	purgeAuth, err := cmd.Flags().GetBool("purge-auth")
	if err != nil {
		return err
	}
	noRevoke, err := cmd.Flags().GetBool("no-revoke")
	if err != nil {
		return err
	}
	timeout, err := cmd.Flags().GetUint32("timeout")
	if err != nil {
		return err
	}
	listenerID := cmd.Flags().Arg(0)
	confirmed, err := common.Confirm(cmd, con, fmt.Sprintf("Retire listener '%s'?", listenerID))
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	reply, err := con.Rpc.RetireListener(con.Context(), &clientpb.ListenerRetire{
		ListenerId:     listenerID,
		PurgeConfig:    purgeConfig,
		PurgeAuth:      purgeAuth,
		NoRevoke:       noRevoke,
		TimeoutSeconds: timeout,
	})
	if err != nil {
		return err
	}
	printForwardListenerStatuses(con, []*clientpb.ForwardListenerStatus{reply})
	return nil
}
