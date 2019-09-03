package remote

import (
	"fmt"
	"strings"
	"text/tabwriter"

	ipb "github.com/mtraver/rpi-ir-remote/irremotepb"
	"github.com/mtraver/rpi-ir-remote/irsend"
)

func String(r ipb.Remote) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Name: %v\n", r.Name)

	w := tabwriter.NewWriter(&b, 0, 0, 1, ' ', 0)

	fmt.Fprintln(w, "Command name\tCode\t")
	for _, code := range r.Code {
		fmt.Fprintln(w, strings.Join([]string{code.Name, code.Code}, "\t")+"\t")
	}

	w.Flush()
	return b.String()
}

func Send(remote ipb.Remote, name string) error {
	var c string
	for _, code := range remote.Code {
		if code.Name == name {
			c = code.Code
		}
	}

	if c == "" {
		return fmt.Errorf("remote: remote %q does not have command %q", remote.Name, name)
	}

	return irsend.Send(remote.Name, c)
}
