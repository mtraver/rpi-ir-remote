package main

import (
	"flag"
	"fmt"
	"os"
	filepath "path"
	"sort"
	"strings"

	"github.com/mtraver/rpi-ir-remote/remote"
	"github.com/mtraver/rpi-ir-remote/remote/cambridgecxacn"
)

var (
	listCommand = flag.NewFlagSet("list", flag.ExitOnError)

	sendCommand = flag.NewFlagSet("send", flag.ExitOnError)
	repeat      int

	remotes = []remote.Remote{cambridgecxacn.New()}
)

func init() {
	listCommand.Usage = func() {
		fmt.Println(`list: list available remotes and their commands
  usage: list`)
		listCommand.PrintDefaults()
	}

	sendCommand.IntVar(&repeat, "repeat", 0, "number of times to repeat command")
	sendCommand.Usage = func() {
		fmt.Println(`send: send an IR code
  usage: send [options] remote command`)
		sendCommand.PrintDefaults()
	}

	flag.Usage = func() {
		fmt.Printf("usage: %s {list,send} [options] [args]\n", filepath.Base(os.Args[0]))
		fmt.Println("\nCommands:")
		listCommand.Usage()
		sendCommand.Usage()
	}
}

func getRemote(name string) (remote.Remote, error) {
	for _, r := range remotes {
		if r.Name == name {
			return r, nil
		}
	}

	return remote.Remote{}, fmt.Errorf("cli: no remote with name %q", name)
}

func list() {
	sort.Slice(remotes, func(i, j int) bool { return remotes[i].Name < remotes[j].Name })

	strs := make([]string, len(remotes))
	for i, r := range remotes {
		strs[i] = strings.TrimRight(fmt.Sprintf("%v", r), "\n")
	}

	fmt.Println(strings.Join(strs, "\n\n"))
}

func send(remoteName, commandName string, repeat int) {
	r, err := getRemote(remoteName)
	if err != nil {
		fmt.Printf("No remote with name %q\n", remoteName)
		os.Exit(2)
	}

	if err := r.Send(commandName); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(2)
	}

	switch subcmd := os.Args[1]; subcmd {
	case "list":
		if err := listCommand.Parse(os.Args[2:]); err == flag.ErrHelp {
			listCommand.Usage()
		}

		list()
	case "send":
		sendCommand.Parse(os.Args[2:])

		if sendCommand.NArg() != 2 {
			sendCommand.Usage()
			os.Exit(2)
		}

		remoteName := sendCommand.Arg(0)
		if remoteName == "" {
			sendCommand.Usage()
			fmt.Println("\nremote must not be the empty string")
			os.Exit(2)
		}

		commandName := sendCommand.Arg(1)
		if commandName == "" {
			sendCommand.Usage()
			fmt.Println("\ncommand must not be the empty string")
			os.Exit(2)
		}

		send(remoteName, commandName, repeat)
	default:
		flag.Usage()
		os.Exit(2)
	}
}
