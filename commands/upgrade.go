package commands

func cmdUpgrade(c MachineCLIClient) error {
	return runActionWithMachineClient("upgrade", c)
}
