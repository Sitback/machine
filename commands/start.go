package commands

import (
	"github.com/docker/machine/libmachine/log"
)

func cmdStart(c MachineCLIClient) error {
	if err := runActionWithMachineClient("start", c); err != nil {
		return err
	}

	log.Info("Started machines may have new IP addresses. You may need to re-run the `docker-machine env` command.")

	return nil
}
