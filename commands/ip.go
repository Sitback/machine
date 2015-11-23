package commands

func cmdIP(c MachineCLIClient) error {
	return runActionWithMachineClient("ip", c)
}
