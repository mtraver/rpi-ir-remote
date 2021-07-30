// Binary makereq creates a JSON-encoded Request proto, which includes an Action and a JWT that authorizes that specific Action.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/golang/protobuf/jsonpb"

	"github.com/mtraver/rpi-ir-remote/auth"
	ipb "github.com/mtraver/rpi-ir-remote/irremotepb"
)

func fatalf(format string, a ...interface{}) {
	fmt.Printf(format+"\n", a...)
	os.Exit(1)
}

func newJWT(deviceID string, action *ipb.Action, keyPath string, ttl time.Duration) (string, error) {
	keyBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return "", err
	}

	key, err := jwt.ParseECPrivateKeyFromPEM(keyBytes)
	if err != nil {
		return "", err
	}

	sha, err := auth.Sum256(action)
	if err != nil {
		return "", err
	}

	token := jwt.New(jwt.SigningMethodES256)
	token.Claims = auth.Claims{
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(ttl).Unix(),
		},
		Device:     deviceID,
		ActionHash: sha,
	}

	return token.SignedString(key)
}

func init() {
	flag.Usage = func() {
		message := `usage: makereq device remote command key_path

makereq creates a JSON-encoded Request proto, which includes
an Action and a JWT that authorizes that specific Action.

Positional arguments (required):
  device
      Google Cloud IoT Core device ID
  remote
      LIRC remote ID
  command
      remote command name
  key_path
      path to file containing the EC private key with which to sign the JWT
`
		fmt.Fprintf(flag.CommandLine.Output(), message)
	}
}

func main() {
	flag.Parse()

	if len(os.Args) != 5 {
		flag.Usage()
		os.Exit(1)
	}

	deviceID := os.Args[1]
	remoteID := os.Args[2]
	command := os.Args[3]
	keyPath := os.Args[4]

	action := &ipb.Action{
		RemoteId: remoteID,
		Command:  command,
	}

	token, err := newJWT(deviceID, action, keyPath, 3650*24*time.Hour)
	if err != nil {
		fatalf("Failed to make JWT: %v", err)
	}

	req := &ipb.Request{
		DeviceId: deviceID,
		Jwt:      token,
		Action:   action,
	}

	marshaler := jsonpb.Marshaler{}
	s, err := marshaler.MarshalToString(req)
	if err != nil {
		fatalf("Failed to marshal request to JSON: %v", err)
	}
	fmt.Println(s)
}
