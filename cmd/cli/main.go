package main

import (
	"flag"
	"fmt"
	"os"
	filepath "path"
	"sort"
	"strings"

	ipb "github.com/mtraver/rpi-ir-remote/irremotepb"
	"github.com/mtraver/rpi-ir-remote/remote"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	listCommand = flag.NewFlagSet("list", flag.ExitOnError)

	sendCommand = flag.NewFlagSet("send", flag.ExitOnError)
	repeat      int

	remotes = []ipb.Remote{}
)

func init() {
	listCommand.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `list: list available remotes and their commands
  usage: list`)
		listCommand.PrintDefaults()
	}

	sendCommand.IntVar(&repeat, "repeat", 0, "number of times to repeat command")
	sendCommand.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `send: send an IR code
  usage: send [options] remote command`)
		sendCommand.PrintDefaults()
	}

	flag.Usage = func() {
		message := `usage: %s remote_proto {list,send} [options] [args]

Positional arguments (required):
  remote_proto
	path to file containing a JSON-encoded remote proto

Commands:
`

		fmt.Fprintf(flag.CommandLine.Output(), message, filepath.Base(os.Args[0]))
		listCommand.Usage()
		sendCommand.Usage()
	}
}

func getRemote(name string) (ipb.Remote, error) {
	for _, r := range remotes {
		if r.Name == name {
			return r, nil
		}
	}

	return ipb.Remote{}, fmt.Errorf("cli: no remote with name %q", name)
}

func list() {
	sort.Slice(remotes, func(i, j int) bool { return remotes[i].Name < remotes[j].Name })

	strs := make([]string, len(remotes))
	for i, r := range remotes {
		strs[i] = strings.TrimRight(remote.String(r), "\n")
	}

	fmt.Println(strings.Join(strs, "\n\n"))
}

func send(remoteName, commandName string, repeat int) {
	r, err := getRemote(remoteName)
	if err != nil {
		fmt.Printf("No remote with name %q\n", remoteName)
		os.Exit(2)
	}

	if err := remote.Send(r, commandName); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 3 {
		flag.Usage()
		os.Exit(2)
	}

	// Parse remote proto.
	rp := os.Args[1]
	rawRP, err := os.ReadFile(rp)
	if err != nil {
		fmt.Printf("Failed to open remote proto %s: %v\n", rp, err)
		os.Exit(1)
	}

	var r ipb.Remote
	if err := protojson.Unmarshal(rawRP, &r); err != nil {
		fmt.Printf("Failed to parse remote proto %s: %v\n", rp, err)
		os.Exit(1)
	}
	remotes = append(remotes, r)

	switch subcmd := os.Args[2]; subcmd {
	case "list":
		if err := listCommand.Parse(os.Args[3:]); err == flag.ErrHelp {
			listCommand.Usage()
		}

		list()
	case "send":
		sendCommand.Parse(os.Args[3:])

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
