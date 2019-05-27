package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
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

	templates = template.Must(template.New("index").Parse(indexTemplate))
)

func init() {
	flag.StringVar(&configFilePath, "config", "", "path to config file")
}

type irRemoteRequest struct {
	Token     string `json:"token"`
	Increment int    `json:"increment"`
}

type irsendHandler struct {
	Remote     remote.Remote
	Config     serverconfig.Config
	Cmd        string
	CheckToken bool
}

func (h irsendHandler) checkToken(r *http.Request) error {
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

		if req.Token != h.Config.Token {
			return fmt.Errorf("irremote: bad token")
		}
	} else {
		return fmt.Errorf("irremote: bad method")
	}

	return nil
}

func (h irsendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.CheckToken {
		if err := h.checkToken(r); err != nil {
			log.Printf("Token check failed: %v", err)
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	log.Printf("Received request: %v %v", h.Remote.Name, h.Cmd)

	if err := h.Remote.Send(h.Cmd); err != nil {
		log.Printf("Failed to send command %q to %q: %v", h.Cmd, h.Remote.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, fmt.Sprintf("%v OK", h.Cmd))
}

type indexHandler struct {
	Remote remote.Remote
	Config serverconfig.Config
}

func (h indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := struct {
		Remote remote.Remote
		Config serverconfig.Config
	}{
		Remote: h.Remote,
		Config: h.Config,
	}

	templates.ExecuteTemplate(w, "index", data)
}

func main() {
	flag.Parse()

	config, err := serverconfig.Load(configFilePath)
	if err != nil {
		log.Printf("Failed to parse config file %v: %v", configFilePath, err)
		os.Exit(1)
	}

	r := cambridgecxacn.New()

	webuiMux := http.NewServeMux()
	webuiMux.Handle("/", indexHandler{
		Remote: r,
		Config: config,
	})

	apiMux := http.NewServeMux()
	for name := range r.Commands {
		apiMux.Handle(fmt.Sprintf("/%v", name), irsendHandler{
			Remote:     r,
			Config:     config,
			Cmd:        name,
			CheckToken: true,
		})

		webuiMux.Handle(fmt.Sprintf("/%v", name), irsendHandler{
			Remote:     r,
			Config:     config,
			Cmd:        name,
			CheckToken: false,
		})
	}

	go func() {
		log.Printf("Web UI server listening on port %v", config.WebUIPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%v", config.WebUIPort), webuiMux); err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}()

	log.Printf("API server listening on port %v", config.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%v", config.Port), apiMux); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
