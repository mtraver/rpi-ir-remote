package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/mtraver/gaelog"
	"github.com/mtraver/iotcore"
	cloudiot "google.golang.org/api/cloudiot/v1"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

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

	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		gaelog.Errorf(ctx, "Failed to read body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var req ipb.Request
	if err := protojson.Unmarshal(body, &req); err != nil {
		gaelog.Errorf(ctx, "Failed to unmarshal Request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.GetAction() == nil || req.GetDeviceId() == "" || req.GetJwt() == "" {
		gaelog.Errorf(ctx, "Action, device ID, and/or JWT empty")
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
		gaelog.Errorf(ctx, "Failed to parse JWT: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	claims, ok := token.Claims.(*auth.Claims)
	if !ok {
		gaelog.Errorf(ctx, "Failed to get claims")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !token.Valid {
		gaelog.Warningf(ctx, "Invalid JWT")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Make sure all custom claims were given.
	if claims.Device == "" {
		gaelog.Warningf(ctx, "Device claim not present or empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if claims.ActionHash == "" {
		gaelog.Warningf(ctx, "ActionHash claim not present or empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Make sure the device in the request matches the claims.
	if req.GetDeviceId() != claims.Device {
		gaelog.Warningf(ctx, "Devices do not match. Request: %q, JWT: %q", req.GetDeviceId(), claims.Device)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Make sure the Action's hash matches the claims.
	action := req.GetAction()
	sha, err := auth.Sum256(action)
	if err != nil {
		gaelog.Errorf(ctx, "Failed to get hash of Action: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if sha != claims.ActionHash {
		gaelog.Warningf(ctx, "Action hashes do not match. Request: %q, JWT: %q", sha, claims.ActionHash)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// The request and JWT are now validated. Send the Action to the device.
	gaelog.Infof(ctx, "Sending Action to %q: %v", claims.Device, action)
	resp, err := sendCommand(iotcore.Device{
		ProjectID:  h.ProjectID,
		RegistryID: h.RegistryID,
		DeviceID:   claims.Device,
		Region:     h.Region,
	}, action)

	success := err == nil && resp.HTTPStatusCode == http.StatusOK
	if !success {
		gaelog.Errorf(ctx, "Failed to send command to device. Error: %q Response: %v", err, resp)
	}

	actionLogMux.Lock()
	defer actionLogMux.Unlock()

	actionLog = append(actionLog, actionRecord{
		Timestamp: time.Now(),
		Success:   success,
		Action:    action,
	})
}

func sendCommand(device iotcore.Device, action *ipb.Action) (*cloudiot.SendCommandToDeviceResponse, error) {
	ctx := context.Background()
	service, err := cloudiot.NewService(ctx, option.WithScopes(cloudiot.CloudiotScope))
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

	return service.Projects.Locations.Registries.Devices.SendCommandToDevice(device.ClientID(), req).Do()
}
