package remote

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/mtraver/rpi-ir-remote/irsend"
)

type Command struct {
	Code           string
	RepeatInterval time.Duration
}

type Remote struct {
	Name     string
	Commands map[string]Command
}

func NewRemote(name string) Remote {
	return Remote{
		Name:     name,
		Commands: make(map[string]Command),
	}
}

func (r Remote) AddCommand(name string, code string) {
	r.Commands[name] = Command{
		Code: code,
	}
}

func (r Remote) Send(command string) error {
	c, ok := r.Commands[command]
	if !ok {
		return fmt.Errorf("remote: remote %q does not have command %q", r.Name, command)
	}

	return irsend.Send(r.Name, c.Code)
}

func (r Remote) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Name: %v\n", r.Name)

	w := tabwriter.NewWriter(&b, 0, 0, 1, ' ', 0)

	fmt.Fprintln(w, "Command name\tCode\t")
	for name, command := range r.Commands {
		fmt.Fprintln(w, strings.Join([]string{name, command.Code}, "\t")+"\t")
	}

	w.Flush()
	return b.String()
}
