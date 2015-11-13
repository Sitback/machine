package commands

func cmdVersion(c MachineCLIClient) error {
	c.ShowVersion()
	return nil
}
