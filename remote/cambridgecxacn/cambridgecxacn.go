package cambridgecxacn

import (
	"github.com/mtraver/rpi-ir-remote/remote"
)

func New() remote.Remote {
	r := remote.NewRemote("cambridge_cxa")

	r.AddCommand("off", "KEY_POWER_OFF")
	r.AddCommand("music", "KEY_SOURCE_D1")
	r.AddCommand("tv", "KEY_SOURCE_D2")
	r.AddCommand("volup", "KEY_VOLUMEUP")
	r.AddCommand("voldown", "KEY_VOLUMEDOWN")
	r.AddCommand("direct", "KEY_DIRECT")
	r.AddCommand("lcd", "KEY_LCD")

	return r
}
