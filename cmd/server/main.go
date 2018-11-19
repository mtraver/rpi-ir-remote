package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	serverconfig "github.com/mtraver/rpi-ir-remote/cmd/server/config"
	"github.com/mtraver/rpi-ir-remote/remote"
	"github.com/mtraver/rpi-ir-remote/remote/cambridgecxacn"
)

const (
	volumeIncMax   = 5
	volumeIncDelay = 0.3
)

var (
	configFilePath string
)

func init() {
	flag.StringVar(&configFilePath, "config", "", "path to config file")
}

type irRemoteRequest struct {
	Token     string `json:"token"`
	Increment int    `json:"increment"`
}

func checkToken(r *http.Request, token string) error {
	if r.Method == "POST" {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}

		var req irRemoteRequest
		err = json.Unmarshal(b, &req)
		if err != nil {
			return err
		}

		if req.Token != token {
			return fmt.Errorf("irremote: bad token")
		}
	} else {
		return fmt.Errorf("irremote: bad method")
	}

	return nil
}

func tokenWrapper(f http.HandlerFunc, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := checkToken(r, token)
		if err != nil {
			log.Printf("Token check failed: %v", err)
			w.WriteHeader(http.StatusForbidden)
			return
		}

		f.ServeHTTP(w, r)
	}
}

func irsendHandler(rmt remote.Remote, cmd string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %v %v", rmt.Name, cmd)

		err := rmt.Send(cmd)
		if err != nil {
			log.Printf("Failed to send command %q to %q: %v", cmd, rmt.Name, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, fmt.Sprintf("%v OK", cmd))
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, "Did you know that the wavelength of infrared radiation ranges from about 800 nm to 1 mm?")
}

func main() {
	flag.Parse()

	config, err := serverconfig.Load(configFilePath)
	if err != nil {
		log.Printf("Failed to parse config file %v: %v", configFilePath, err)
		os.Exit(1)
	}

	r := cambridgecxacn.New()

	http.HandleFunc("/", indexHandler)
	for name := range r.Commands {
		http.HandleFunc(fmt.Sprintf("/%v", name), tokenWrapper(irsendHandler(r, name), config.Token))
	}

	log.Printf("Listening on port %v", config.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%v", config.Port), nil); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
