package lume

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/shell-local/localexec"
)

const lumeCommand = "lume"

func PathInLumeHome(elem ...string) string {
	if home := os.Getenv("LUME_HOME"); home != "" {
		return path.Join(home, path.Join(elem...))
	}
	userHome, _ := os.UserHomeDir()
	return path.Join(userHome, ".lume", path.Join(elem...))
}

type execBuilder struct {
	ctx           context.Context
	args          []string
	sleepDuration int64
	ui            packer.Ui
}

func LumeExec() *execBuilder {
	return &execBuilder{}
}

func (eb *execBuilder) WithSleep(durationSeconds int64) *execBuilder {
	eb.sleepDuration = durationSeconds
	return eb
}

func (eb *execBuilder) WithPackerUI(ui packer.Ui) *execBuilder {
	eb.ui = ui
	return eb
}

func (eb *execBuilder) WithContext(ctx context.Context) *execBuilder {
	eb.ctx = ctx
	return eb
}

func (eb *execBuilder) WithArgs(args ...string) *execBuilder {
	eb.args = append(eb.args, args...)
	return eb
}

func (eb *execBuilder) Do() (string, error) {

	var cmd *exec.Cmd
	if eb.sleepDuration != 0 {
		// this parses to 'lume get vm <vm-id>' as an example
		lumeCmdArgs := append([]string{lumeCommand}, eb.args...)
		lumeCmdString := strings.Join(lumeCmdArgs, " ")

		// sleep command parses to 'sleep 2' as an example
		sleepCmdString := fmt.Sprintf("sleep %v", eb.sleepDuration)

		// complete command parses to 'sleep 2 && lume get vm <vm-id>' as an example
		completeCmdString := fmt.Sprintf("%v && %v", sleepCmdString, lumeCmdString)
		cmd = exec.CommandContext(eb.ctx, "/bin/bash", "-c", completeCmdString)
	} else {
		cmd = exec.CommandContext(eb.ctx, lumeCommand, eb.args...)
	}

	if eb.ui != nil {
		return "", localexec.RunAndStream(cmd, eb.ui, []string{})
	} else {
		log.Printf("Executing lume: %#v", eb.args)

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()

		outString := strings.TrimSpace(out.String())

		if _, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("lume error: %s", outString)
		}

		return outString, err
	}
}
