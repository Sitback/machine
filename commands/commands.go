package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/machine/cli"
	"github.com/docker/machine/commands/mcndirs"
	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/cert"
	"github.com/docker/machine/libmachine/host"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/mcnutils"
	"github.com/docker/machine/libmachine/persist"
	"github.com/docker/machine/libmachine/ssh"
)

var (
	ErrUnknownShell       = errors.New("Error: Unknown shell")
	ErrNoMachineSpecified = errors.New("Error: Expected to get one or more machine names as arguments")
	ErrExpectedOneMachine = errors.New("Error: Expected one machine name as an argument")
)

// CommandLine contains all the information passed to the commands on the command line.
type CommandLine interface {
	ShowHelp()

	ShowVersion()

	Application() *cli.App

	Args() cli.Args

	Bool(name string) bool

	String(name string) string

	StringSlice(name string) []string

	GlobalString(name string) string

	FlagNames() (names []string)

	Generic(name string) interface{}
}

// MachineCLIClient embeds CommandLine (codegangsta/cli interaction) and
// libmachine.API (libmachine interaction) interfaces, and a method to get the
// Store directly.
type MachineCLIClient interface {
	CommandLine
	libmachine.API
}

type contextCommandLine struct {
	*cli.Context
	libmachine.API
}

func (c *contextCommandLine) ShowHelp() {
	cli.ShowCommandHelp(c.Context, c.Command.Name)
}

func (c *contextCommandLine) ShowVersion() {
	cli.ShowVersion(c.Context)
}

func (c *contextCommandLine) Application() *cli.App {
	return c.App
}

func runActionWithMachineClient(actionName string, c MachineCLIClient) error {
	hosts, err := persist.LoadHosts(c, c.Args())
	if err != nil {
		return err
	}

	if len(hosts) == 0 {
		return ErrNoMachineSpecified
	}

	if errs := runActionForeachMachine(actionName, hosts); len(errs) > 0 {
		return consolidateErrs(errs)
	}

	for _, h := range hosts {
		if err := c.Save(h); err != nil {
			return fmt.Errorf("Error saving host to store: %s", err)
		}
	}

	return nil
}

func fatalOnError(command func(commandLine MachineCLIClient) error) func(context *cli.Context) {
	return func(context *cli.Context) {
		client := libmachine.NewClient(mcndirs.GetBaseDir())

		if context.GlobalBool("native-ssh") {
			client.SSHClientType = ssh.Native
		}
		client.GithubAPIToken = context.GlobalString("github-api-token")
		client.Filestore.Path = context.GlobalString("storage-path")

		// TODO (nathanleclaire): These should ultimately be accessed
		// through the libmachine client by the rest of the code and
		// not through their respective modules.  For now, however,
		// they are also being set the way that they originally were
		// set to preserve backwards compatibility.
		mcndirs.BaseDir = client.Filestore.Path
		mcnutils.GithubAPIToken = client.GithubAPIToken
		ssh.SetDefaultClient(client.SSHClientType)

		if err := command(&contextCommandLine{context, client}); err != nil {
			log.Fatal(err)
		}
	}
}

func confirmInput(msg string) (bool, error) {
	fmt.Printf("%s (y/n): ", msg)

	var resp string
	_, err := fmt.Scanln(&resp)
	if err != nil {
		return false, err
	}

	confirmed := strings.Index(strings.ToLower(resp), "y") == 0
	return confirmed, nil
}

var Commands = []cli.Command{
	{
		Name:   "active",
		Usage:  "Print which machine is active",
		Action: fatalOnError(cmdActive),
	},
	{
		Name:        "config",
		Usage:       "Print the connection config for machine",
		Description: "Argument is a machine name.",
		Action:      fatalOnError(cmdConfig),
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "swarm",
				Usage: "Display the Swarm config instead of the Docker daemon",
			},
		},
	},
	{
		Flags:           sharedCreateFlags,
		Name:            "create",
		Usage:           "Create a machine",
		Description:     fmt.Sprintf("Run '%s create --driver name' to include the create flags for that driver in the help text.", os.Args[0]),
		Action:          fatalOnError(cmdCreateOuter),
		SkipFlagParsing: true,
	},
	{
		Name:        "env",
		Usage:       "Display the commands to set up the environment for the Docker client",
		Description: "Argument is a machine name.",
		Action:      fatalOnError(cmdEnv),
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "swarm",
				Usage: "Display the Swarm config instead of the Docker daemon",
			},
			cli.StringFlag{
				Name:  "shell",
				Usage: "Force environment to be configured for specified shell",
			},
			cli.BoolFlag{
				Name:  "unset, u",
				Usage: "Unset variables instead of setting them",
			},
			cli.BoolFlag{
				Name:  "no-proxy",
				Usage: "Add machine IP to NO_PROXY environment variable",
			},
		},
	},
	{
		Name:        "inspect",
		Usage:       "Inspect information about a machine",
		Description: "Argument is a machine name.",
		Action:      fatalOnError(cmdInspect),
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "format, f",
				Usage: "Format the output using the given go template.",
				Value: "",
			},
		},
	},
	{
		Name:        "ip",
		Usage:       "Get the IP address of a machine",
		Description: "Argument(s) are one or more machine names.",
		Action:      fatalOnError(cmdIP),
	},
	{
		Name:        "kill",
		Usage:       "Kill a machine",
		Description: "Argument(s) are one or more machine names.",
		Action:      fatalOnError(cmdKill),
	},
	{
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "quiet, q",
				Usage: "Enable quiet mode",
			},
			cli.StringSliceFlag{
				Name:  "filter",
				Usage: "Filter output based on conditions provided",
				Value: &cli.StringSlice{},
			},
		},
		Name:   "ls",
		Usage:  "List machines",
		Action: fatalOnError(cmdLs),
	},
	{
		Name:        "regenerate-certs",
		Usage:       "Regenerate TLS Certificates for a machine",
		Description: "Argument(s) are one or more machine names.",
		Action:      fatalOnError(cmdRegenerateCerts),
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "force, f",
				Usage: "Force rebuild and do not prompt",
			},
		},
	},
	{
		Name:        "restart",
		Usage:       "Restart a machine",
		Description: "Argument(s) are one or more machine names.",
		Action:      fatalOnError(cmdRestart),
	},
	{
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "force, f",
				Usage: "Remove local configuration even if machine cannot be removed",
			},
		},
		Name:        "rm",
		Usage:       "Remove a machine",
		Description: "Argument(s) are one or more machine names.",
		Action:      fatalOnError(cmdRm),
	},
	{
		Name:            "ssh",
		Usage:           "Log into or run a command on a machine with SSH.",
		Description:     "Arguments are [machine-name] [command]",
		Action:          fatalOnError(cmdSSH),
		SkipFlagParsing: true,
	},
	{
		Name:        "scp",
		Usage:       "Copy files between machines",
		Description: "Arguments are [machine:][path] [machine:][path].",
		Action:      fatalOnError(cmdScp),
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "recursive, r",
				Usage: "Copy files recursively (required to copy directories)",
			},
		},
	},
	{
		Name:        "start",
		Usage:       "Start a machine",
		Description: "Argument(s) are one or more machine names.",
		Action:      fatalOnError(cmdStart),
	},
	{
		Name:        "status",
		Usage:       "Get the status of a machine",
		Description: "Argument is a machine name.",
		Action:      fatalOnError(cmdStatus),
	},
	{
		Name:        "stop",
		Usage:       "Stop a machine",
		Description: "Argument(s) are one or more machine names.",
		Action:      fatalOnError(cmdStop),
	},
	{
		Name:        "upgrade",
		Usage:       "Upgrade a machine to the latest version of Docker",
		Description: "Argument(s) are one or more machine names.",
		Action:      fatalOnError(cmdUpgrade),
	},
	{
		Name:        "url",
		Usage:       "Get the URL of a machine",
		Description: "Argument is a machine name.",
		Action:      fatalOnError(cmdURL),
	},
	{
		Name:   "version",
		Usage:  "Show the Docker Machine version information",
		Action: fatalOnError(cmdVersion),
	},
}

func printIP(h *host.Host) func() error {
	return func() error {
		ip, err := h.Driver.GetIP()
		if err != nil {
			return fmt.Errorf("Error getting IP address: %s", err)
		}
		fmt.Println(ip)
		return nil
	}
}

// machineCommand maps the command name to the corresponding machine command.
// We run commands concurrently and communicate back an error if there was one.
func machineCommand(actionName string, host *host.Host, errorChan chan<- error) {
	// TODO: These actions should have their own type.
	commands := map[string](func() error){
		"configureAuth": host.ConfigureAuth,
		"start":         host.Start,
		"stop":          host.Stop,
		"restart":       host.Restart,
		"kill":          host.Kill,
		"upgrade":       host.Upgrade,
		"ip":            printIP(host),
	}

	log.Debugf("command=%s machine=%s", actionName, host.Name)

	errorChan <- commands[actionName]()
}

// runActionForeachMachine will run the command across multiple machines
func runActionForeachMachine(actionName string, machines []*host.Host) []error {
	var (
		numConcurrentActions = 0
		errorChan            = make(chan error)
		errs                 = []error{}
	)

	for _, machine := range machines {
		numConcurrentActions++
		go machineCommand(actionName, machine, errorChan)
	}

	// TODO: We should probably only do 5-10 of these
	// at a time, since otherwise cloud providers might
	// rate limit us.
	for i := 0; i < numConcurrentActions; i++ {
		if err := <-errorChan; err != nil {
			errs = append(errs, err)
		}
	}

	close(errorChan)

	return errs
}

func consolidateErrs(errs []error) error {
	finalErr := ""
	for _, err := range errs {
		finalErr = fmt.Sprintf("%s\n%s", finalErr, err)
	}

	return errors.New(strings.TrimSpace(finalErr))
}

// Returns the cert paths.  codegangsta/cli will not set the cert paths if the
// storage-path is set to something different so we cannot use the paths in the
// global options. le sigh.
func getCertPathInfoFromCommandLine(c CommandLine) cert.PathInfo {
	caCertPath := c.GlobalString("tls-ca-cert")
	caKeyPath := c.GlobalString("tls-ca-key")
	clientCertPath := c.GlobalString("tls-client-cert")
	clientKeyPath := c.GlobalString("tls-client-key")

	if caCertPath == "" {
		caCertPath = filepath.Join(mcndirs.GetMachineCertDir(), "ca.pem")
	}

	if caKeyPath == "" {
		caKeyPath = filepath.Join(mcndirs.GetMachineCertDir(), "ca-key.pem")
	}

	if clientCertPath == "" {
		clientCertPath = filepath.Join(mcndirs.GetMachineCertDir(), "cert.pem")
	}

	if clientKeyPath == "" {
		clientKeyPath = filepath.Join(mcndirs.GetMachineCertDir(), "key.pem")
	}

	return cert.PathInfo{
		CaCertPath:       caCertPath,
		CaPrivateKeyPath: caKeyPath,
		ClientCertPath:   clientCertPath,
		ClientKeyPath:    clientKeyPath,
	}
}
