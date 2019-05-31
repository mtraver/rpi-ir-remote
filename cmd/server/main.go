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
	deviceFilePath string
	caCerts        string

	templates = template.Must(template.New("index").Parse(indexTemplate))
)

func init() {
	flag.StringVar(&configFilePath, "config", "", "path to config file")
	flag.StringVar(&deviceFilePath, "device", "", "path to Google Cloud IoT core device config file")
	flag.StringVar(&caCerts, "cacerts", "", "Path to a set of trustworthy CA certs.\nDownload Google's from https://pki.google.com/roots.pem.")
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
		Remote  remote.Remote
		Config  serverconfig.Config
		FunFact string
	}{
		Remote:  h.Remote,
		Config:  h.Config,
		FunFact: funFacts[rand.Intn(len(funFacts))],
	}

	templates.ExecuteTemplate(w, "index", data)
}

func commandHandler(client mqtt.Client, msg mqtt.Message) {
	// TODO(mtraver) Actually handle the message.
	log.Printf("commandHandler: topic: %q, payload: %v", msg.Topic(), msg.Payload())
	msg.Ack()
}

func mqttConnect(device iotcore.Device) (mqtt.Client, error) {
	certsFile, err := os.Open(caCerts)
	if err != nil {
		return nil, err
	}
	defer certsFile.Close()

	client, err := device.NewClient(iotcore.DefaultBroker, certsFile, iotcore.CacheJWT(60*time.Minute))
	if err != nil {
		return nil, fmt.Errorf("Failed to make MQTT client: %v", err)
	}

	// Connect to the MQTT server.
	waitDur := 10 * time.Second
	token := client.Connect()
	if ok := token.WaitTimeout(waitDur); !ok {
		return nil, fmt.Errorf("MQTT connection attempt timed out after %v", waitDur)
	} else if token.Error() != nil {
		return nil, fmt.Errorf("Failed to connect to MQTT server: %v", token.Error())
	}

	// Subscribe to the command topic.
	token = client.Subscribe(device.CommandTopic(), 1, commandHandler)
	if ok := token.WaitTimeout(waitDur); !ok {
		return nil, fmt.Errorf("Subscription attempt to command topic %s timed out after %v", device.CommandTopic(), waitDur)
	} else if token.Error() != nil {
		return nil, fmt.Errorf("Failed to subscribe to command topic %s: %v", device.CommandTopic(), token.Error())
	}

	return client, nil
}

func main() {
	flag.Parse()
	if deviceFilePath != "" && caCerts == "" {
		fmt.Fprintf(flag.CommandLine.Output(), "-cacerts is required when -device is given\n")
		flag.Usage()
		os.Exit(2)
	}

	config, err := serverconfig.Load(configFilePath)
	if err != nil {
		log.Fatalf("Failed to parse config file %s: %v", configFilePath, err)
	}

	// If an MQTT device config is given, connect to MQTT and subscribe to the command topic.
	if deviceFilePath != "" {
		device, err := parseDeviceConfig(deviceFilePath)
		if err != nil {
			log.Fatalf("Failed to parse device config file: %v", err)
		}

		client, err := mqttConnect(device)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Connected to MQTT broker")

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

	// TODO(mtraver) If we're using MQTT we shouldn't start the API HTTP server.
	go func() {
		log.Printf("API server listening on port %v", config.Port)
		if err := http.ListenAndServe(fmt.Sprintf(":%v", config.Port), apiMux); err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}()

	log.Printf("Web UI server listening on port %v", config.WebUIPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%v", config.WebUIPort), webuiMux); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
