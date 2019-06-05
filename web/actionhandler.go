package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/mtraver/gaelog"
	"github.com/mtraver/iotcore"
	"golang.org/x/oauth2/google"
	cloudiot "google.golang.org/api/cloudiot/v1"

	"github.com/mtraver/rpi-ir-remote/auth"
	ipb "github.com/mtraver/rpi-ir-remote/irremotepb"
)

type actionHandler struct {
	ProjectID  string
	RegistryID string
	Region     string
	PublicKey  *ecdsa.PublicKey
}

func (h actionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	lg, err := gaelog.New(r)
	if err != nil {
		lg.Errorf("%v", err)
	}
	defer lg.Close()

	var req ipb.Request
	err = jsonpb.Unmarshal(r.Body, &req)
	defer r.Body.Close()
	if err != nil {
		lg.Errorf("Failed to unmarshal Request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.GetAction() == nil || req.GetDeviceId() == "" || req.GetJwt() == "" {
		lg.Errorf("Action, device ID, and/or JWT empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := jwt.ParseWithClaims(req.GetJwt(), &auth.Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing algorithm.
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
		}

		return h.PublicKey, nil
	})

	if err != nil {
		lg.Errorf("Failed to parse JWT: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	claims, ok := token.Claims.(*auth.Claims)
	if !ok {
		lg.Errorf("Failed to get claims")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !token.Valid {
		lg.Warningf("Invalid JWT")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Make sure all custom claims were given.
	if claims.Device == "" {
		lg.Warningf("Device claim not present or empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if claims.ActionHash == "" {
		lg.Warningf("ActionHash claim not present or empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Make sure the device in the request matches the claims.
	if req.GetDeviceId() != claims.Device {
		lg.Warningf("Devices do not match. Request: %q, JWT: %q", req.GetDeviceId(), claims.Device)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Make sure the Action's hash matches the claims.
	action := req.GetAction()
	sha, err := auth.Sum256(action)
	if err != nil {
		lg.Errorf("Failed to get hash of Action: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if sha != claims.ActionHash {
		lg.Warningf("Action hashes do not match. Request: %q, JWT: %q", sha, claims.ActionHash)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// The request and JWT are now validated. Send the Action to the device.
	d := iotcore.Device{
		ProjectID:  h.ProjectID,
		RegistryID: h.RegistryID,
		DeviceID:   claims.Device,
		Region:     h.Region,
	}

	resp, err := sendCommand(d, action)
	lg.Infof("SendCommandToDeviceResponse: %v", resp)
	if err != nil {
		lg.Errorf("Failed to send command to device: %v", err)
	}
}

func sendCommand(device iotcore.Device, action *ipb.Action) (*cloudiot.SendCommandToDeviceResponse, error) {
	ctx := context.Background()
	httpClient, err := google.DefaultClient(ctx, cloudiot.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	client, err := cloudiot.New(httpClient)
	if err != nil {
		return nil, err
	}

	data, err := proto.Marshal(action)
	if err != nil {
		return nil, err
	}
	req := &cloudiot.SendCommandToDeviceRequest{
		BinaryData: base64.StdEncoding.EncodeToString(data),
	}

	return client.Projects.Locations.Registries.Devices.SendCommandToDevice(device.ClientID(), req).Do()
}
