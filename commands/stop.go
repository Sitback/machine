package commands

func cmdStop(c MachineCLIClient) error {
	return runActionWithMachineClient("stop", c)
}
