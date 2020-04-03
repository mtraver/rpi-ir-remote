package main

import (
	"crypto/ecdsa"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/mtraver/gaelog"
	ipb "github.com/mtraver/rpi-ir-remote/irremotepb"
)

type actionRecord struct {
	Timestamp time.Time
	Success   bool
	Action    *ipb.Action
}

var (
	actionLog    = make([]actionRecord, 0, 16)
	actionLogMux sync.Mutex
)

func mustGetenv(varName string) string {
	val := os.Getenv(varName)
	if val == "" {
		log.Fatalf("Environment variable must be set: %v\n", varName)
	}
	return val
}

func mustParseKey(filePath string) *ecdsa.PublicKey {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read public key file: %v", err)
	}

	key, err := jwt.ParseECPublicKeyFromPEM(b)
	if err != nil {
		log.Fatalf("Failed to parse EC public key from PEM: %v", err)
	}
	return key
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "ok")
}

func main() {
	templates := template.Must(template.New("status.html").Funcs(
		template.FuncMap{
			"RFC3339": func(t time.Time) string {
				return t.Format(time.RFC3339)
			},
		}).ParseGlob("web/templates/*"))

	mux := http.NewServeMux()

	mux.HandleFunc("/", rootHandler)
	mux.Handle("/action", actionHandler{
		ProjectID:  mustGetenv("GOOGLE_CLOUD_PROJECT"),
		RegistryID: mustGetenv("IOTCORE_REGISTRY"),
		Region:     mustGetenv("IOTCORE_REGION"),
		PublicKey:  mustParseKey(mustGetenv("PUB_KEY_PATH")),
	})
	mux.Handle("/status", statusHandler{
		Template: templates,
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), gaelog.Wrap(mux)))
}
