package commands

func cmdKill(c MachineCLIClient) error {
	return runActionWithMachineClient("kill", c)
}
