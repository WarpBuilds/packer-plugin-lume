package lume

import (
	"context"
)

func TartMachineIP(ctx context.Context, vmName string, ipExtraArgs []string) (string, error) {
	ipArgs := []string{"ip", "--wait", "120", vmName}
	if len(ipExtraArgs) > 0 {
		ipArgs = append(ipArgs, ipExtraArgs...)
	}
	return LumeExec().WithContext(ctx).
		WithSleep(120).
		WithArgs(ipArgs...).Do()
}
