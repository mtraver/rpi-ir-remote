package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mtraver/iotcore"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	serverconfig "github.com/mtraver/rpi-ir-remote/cmd/server/config"
	cpb "github.com/mtraver/rpi-ir-remote/cmd/server/configpb"
	ipb "github.com/mtraver/rpi-ir-remote/irremotepb"
	"github.com/mtraver/rpi-ir-remote/remote"
)

const (
	defaultAPIPort   = 9090
	defaultWebUIPort = 8080

	volumeIncMax   = 5
	volumeIncDelay = 0.3
)

var (
	configFilePath string
	deviceFilePath string
	caCerts        string

	remotes = make(map[string]ipb.Remote)

	// version is the version of the binary, which should be set on the command line
	// at build time: go build -ldflags "-X main.version=version-name-here" ...
	version string
)

func init() {
	flag.StringVar(&configFilePath, "config", "", "path to config file")
	flag.StringVar(&deviceFilePath, "device", "", "path to a file containing a JSON-encoded Device struct (see github.com/mtraver/iotcore)")
	flag.StringVar(&caCerts, "cacerts", "", "Path to a set of trustworthy CA certs.\nDownload Google's from https://pki.google.com/roots.pem.")

	flag.Usage = func() {
		message := `usage: server [options] remote_proto [remote_proto [remote_proto ...]]:

Version %s

Positional arguments (required):
  remote_proto
	path to file containing a JSON-encoded remote proto

Options:
`

		v := version
		if v == "" {
			v = "unknown"
		}

		fmt.Fprintf(flag.CommandLine.Output(), message, v)
		flag.PrintDefaults()
	}
}

type irRemoteRequest struct {
	Token     string `json:"token"`
	Increment int    `json:"increment"`
}

type irsendHandler struct {
	Remote     ipb.Remote
	Config     cpb.Config
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

		if req.Token != h.Config.GetToken() {
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

	if err := remote.Send(h.Remote, h.Cmd); err != nil {
		log.Printf("Failed to send command %q to %q: %v", h.Cmd, h.Remote.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, fmt.Sprintf("%v OK", h.Cmd))
}

type indexHandler struct {
	Templates *template.Template
	Remotes   map[string]ipb.Remote
	Config    cpb.Config
}

func (h indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	v := version
	if v == "" {
		v = "unknown"
	}

	data := struct {
		Remotes map[string]ipb.Remote
		Config  cpb.Config
		FunFact string
		Version string
	}{
		Remotes: h.Remotes,
		Config:  h.Config,
		FunFact: funFacts[rand.Intn(len(funFacts))],
		Version: v,
	}

	h.Templates.ExecuteTemplate(w, "index", data)
}

func commandHandler(client mqtt.Client, msg mqtt.Message) {
	msg.Ack()

	var action ipb.Action
	err := proto.Unmarshal(msg.Payload(), &action)
	if err != nil {
		log.Printf("commandHandler: failed to unmarshal Action: %v", err)
		return
	}

	log.Printf("commandHandler: topic: %q, action: %v", msg.Topic(), action)

	if action.GetRemoteId() == "" || action.GetCommand() == "" {
		log.Printf("commandHandler: remote ID and/or command empty")
		return
	}

	r, ok := remotes[action.GetRemoteId()]
	if !ok {
		log.Printf("commandHandler: no remote with ID %q", action.GetRemoteId())
		return
	}

	if err := remote.Send(r, action.GetCommand()); err != nil {
		log.Printf("commandHandler: failed to send command %q to %q: %v", action.GetCommand(), r.Name, err)
		return
	}
}

func onConnect(device *iotcore.Device, opts *mqtt.ClientOptions) error {
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Printf("Connected to MQTT broker")

		// Subscribe to the command topic.
		topic := device.CommandTopic()
		waitDur := 10 * time.Second
		if token := client.Subscribe(topic, 1, commandHandler); !token.WaitTimeout(waitDur) {
			log.Printf("Subscription attempt to command topic %s timed out after %v", topic, waitDur)
		} else if token.Error() != nil {
			log.Printf("Failed to subscribe to command topic %s: %v", topic, token.Error())
		} else {
			log.Printf("Subscribed to command topic %s", topic)
		}
	})
	return nil
}

func onConnectionLost(device *iotcore.Device, opts *mqtt.ClientOptions) error {
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("Connection to MQTT broker lost: %v", err)
	})
	return nil
}

func mqttConnect(device iotcore.Device) (mqtt.Client, error) {
	certsFile, err := os.Open(caCerts)
	if err != nil {
		return nil, err
	}
	defer certsFile.Close()

	client, err := device.NewClient(iotcore.DefaultBroker, certsFile, iotcore.CacheJWT(60*time.Minute), onConnect, onConnectionLost)
	if err != nil {
		return nil, fmt.Errorf("Failed to make MQTT client: %v", err)
	}

	// Connect to the MQTT server.
	waitDur := 10 * time.Second
	if token := client.Connect(); !token.WaitTimeout(waitDur) {
		return nil, fmt.Errorf("MQTT connection attempt timed out after %v", waitDur)
	} else if token.Error() != nil {
		return nil, fmt.Errorf("Failed to connect to MQTT broker: %v", token.Error())
	}

	return client, nil
}

func main() {
	templates := template.Must(template.New("index").Funcs(
		map[string]interface{}{
			"add": func(a, b int) int {
				return a + b
			},
			"sub": func(a, b int) int {
				return a - b
			},
		}).Parse(indexTemplate))

	flag.Parse()
	if deviceFilePath != "" && caCerts == "" {
		fmt.Fprintf(flag.CommandLine.Output(), "-cacerts is required when -device is given\n")
		flag.Usage()
		os.Exit(2)
	}

	if len(flag.Args()) < 1 {
		fmt.Fprintf(flag.CommandLine.Output(), "At least one remote proto must be given\n")
		flag.Usage()
		os.Exit(2)
	}

	// Parse config.
	config, err := serverconfig.Load(configFilePath)
	if err != nil {
		log.Fatalf("Failed to parse config file %s: %v", configFilePath, err)
	}

	// Parse remote protos.
	for _, rp := range flag.Args() {
		rawRP, err := os.ReadFile(rp)
		if err != nil {
			log.Fatalf("Failed to read remote proto %s: %v", rp, err)
		}

		var r ipb.Remote
		if err := protojson.Unmarshal(rawRP, &r); err != nil {
			log.Fatalf("Failed to parse remote proto %s: %v", rp, err)
		}
		remotes[r.Name] = r
	}

	// If an MQTT device config was given, connect to the MQTT broker. In the connect handler
	// we'll subscribe to the commands topic.
	if deviceFilePath != "" {
		device, err := parseDeviceConfig(deviceFilePath)
		if err != nil {
			log.Fatalf("Failed to parse device config file: %v", err)
		}

		client, err := mqttConnect(device)
		if err != nil {
			log.Fatal(err)
		}

		// If the program is killed, disconnect from the MQTT server.
		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			log.Println("Cleaning up...")
			client.Disconnect(250)
			time.Sleep(500 * time.Millisecond)
			os.Exit(1)
		}()
	}

	webuiMux := http.NewServeMux()
	webuiMux.Handle("/", indexHandler{
		Templates: templates,
		Remotes:   remotes,
		Config:    config,
	})

	apiMux := http.NewServeMux()
	for _, r := range remotes {
		for _, code := range r.Code {
			apiMux.Handle(fmt.Sprintf("/%v/%v", r.Name, code.Name), irsendHandler{
				Remote:     r,
				Config:     config,
				Cmd:        code.Name,
				CheckToken: true,
			})

			webuiMux.Handle(fmt.Sprintf("/%v/%v", r.Name, code.Name), irsendHandler{
				Remote:     r,
				Config:     config,
				Cmd:        code.Name,
				CheckToken: false,
			})
		}
	}

	// If an MQTT device config was not given, start the API HTTP server.
	if deviceFilePath == "" {
		go func() {
			port := config.GetPort()
			if port == 0 {
				port = defaultAPIPort
			}

			log.Printf("API server listening on port %v", port)
			if err := http.ListenAndServe(fmt.Sprintf(":%v", port), apiMux); err != nil {
				log.Println(err)
				os.Exit(1)
			}
		}()
	}

	port := config.GetWebuiPort()
	if port == 0 {
		port = defaultWebUIPort
	}

	log.Printf("Web UI server listening on port %v", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%v", port), webuiMux); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
