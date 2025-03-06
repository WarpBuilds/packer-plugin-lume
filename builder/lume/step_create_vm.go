package lume

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCreateVM struct{}

func (s *stepCreateVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Creating virtual machine...")

	createArguments := []string{"create"}
	// ?
	// bash-3.2$ lume create -h
	// Error: Missing expected argument '<name>'
	// Help:  <name>  Name for the virtual machine
	// Usage: lume create <name> [--os <os>] [--cpu <cpu>] [--memory <memory>] [--disk-size <disk-size>] [--display <display>] [--ipsw <ipsw>]
	//   See 'lume create --help' for more information.
	if config.IPSW != "" {
		createArguments = append(createArguments, "--ipsw", config.IPSW)
	}
	if config.CpuCount != 0 {
		createArguments = append(createArguments, "--cpu", strconv.Itoa(int(config.CpuCount)))
	}
	if config.MemoryMb != 0 {
		createArguments = append(createArguments, "--memory", strconv.Itoa(int(config.MemoryMb)))
	}
	if config.DiskSizeGb > 0 {
		createArguments = append(createArguments, "--disk-size", strconv.Itoa(int(config.DiskSizeGb)))
	}

	createArguments = append(createArguments, config.VMName)

	if _, err := LumeExec().WithContext(ctx).WithPackerUI(ui).WithArgs(createArguments...).Do(); err != nil {
		err := fmt.Errorf("Failed to create a VM: %s", err)
		state.Put("error", err)
		return multistep.ActionHalt
	}

	state.Put("vm_name", config.VMName)

	if config.CreateGraceTime != 0 {
		message := fmt.Sprintf("Waiting %v to let the Virtualization.Framework's installation process "+
			"to finish correctly...", config.CreateGraceTime)
		ui.Say(message)
		time.Sleep(config.CreateGraceTime)
	}

	return multistep.ActionContinue
}

func (s *stepCreateVM) Cleanup(state multistep.StateBag) {
	// nothing to clean up
}
