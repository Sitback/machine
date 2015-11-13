package commands

import (
	"github.com/docker/machine/libmachine/log"
)

func cmdRestart(c MachineCLIClient) error {
	if err := runActionWithMachineClient("restart", c); err != nil {
		return err
	}

	log.Info("Restarted machines may have new IP addresses. You may need to re-run the `docker-machine env` command.")

	return nil
}
