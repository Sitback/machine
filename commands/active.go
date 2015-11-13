package commands

import (
	"errors"
	"fmt"

	"github.com/docker/machine/libmachine/persist"
)

var (
	errTooManyArguments = errors.New("Error: Too many arguments given")
	errNoActiveHost     = errors.New("No active host found")
)

func cmdActive(c MachineCLIClient) error {
	if len(c.Args()) > 0 {
		return errTooManyArguments
	}

	hosts, err := persist.LoadAllHosts(c)
	if err != nil {
		return fmt.Errorf("Error getting active host: %s", err)
	}

	items := getHostListItems(hosts)

	for _, item := range items {
		if item.Active {
			fmt.Println(item.Name)
			return nil
		}
	}

	return errNoActiveHost
}
