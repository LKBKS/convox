package cli

import (
	"fmt"
	"os"

	"github.com/convox/convox/pkg/common"
	"github.com/convox/convox/pkg/options"
	"github.com/convox/convox/pkg/structs"
	"github.com/convox/convox/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("test", "run tests", Test, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagApp,
			flagRack,
			stdcli.StringFlag("description", "d", "description"),
			stdcli.StringFlag("release", "", "use existing release to run tests"),
			stdcli.IntFlag("timeout", "t", "timeout"),
		},
		Usage:    "[dir]",
		Validate: stdcli.ArgsMax(1),
	}, WithCloud())
}

func Test(rack sdk.Interface, c *stdcli.Context) error {
	release := c.String("release")

	if release == "" {
		b, err := build(rack, c, true)
		if err != nil {
			return err
		}

		release = b.Release
	}

	m, _, err := common.ReleaseManifest(rack, app(c), release)
	if err != nil {
		return err
	}

	timeout := 3600

	if t := c.Int("timeout"); t > 0 {
		timeout = t
	}

	for _, s := range m.Services {
		if s.Test == "" {
			continue
		}

		c.Writef("Running <command>%s</command> on <service>%s</service>\n", s.Test, s.Name)

		ropts := structs.ProcessRunOptions{
			Command: options.String(fmt.Sprintf("sleep %d", timeout)),
			Release: options.String(release),
		}

		ps, err := rack.ProcessRun(app(c), s.Name, ropts)
		if err != nil {
			return err
		}

		defer rack.ProcessStop(app(c), ps.Id)

		if err := common.WaitForProcessRunning(rack, c, app(c), ps.Id); err != nil {
			return err
		}

		eopts := structs.ProcessExecOptions{
			Entrypoint: options.Bool(true),
		}

		if w, h, err := c.TerminalSize(); err == nil {
			eopts.Height = options.Int(h)
			eopts.Width = options.Int(w)
		}

		if !stdcli.IsTerminal(os.Stdin) {
			eopts.Tty = options.Bool(false)
		}

		code, err := rack.ProcessExec(app(c), ps.Id, s.Test, c, eopts)
		if err != nil {
			return err
		}

		if code != 0 {
			return fmt.Errorf("exit %d", code)
		}
	}

	return nil
}
