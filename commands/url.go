package commands

import (
	"fmt"
)

func cmdURL(c MachineCLIClient) error {
	if len(c.Args()) != 1 {
		return ErrExpectedOneMachine
	}

	host, err := c.Load(c.Args().First())
	if err != nil {
		return err
	}

	url, err := host.GetURL()
	if err != nil {
		return err
	}

	fmt.Println(url)

	return nil
}
