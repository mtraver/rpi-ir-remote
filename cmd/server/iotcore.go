package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mtraver/iotcore"
)

const (
	certExtension = ".x509"

	// TODO(mtraver) Get home dir programmatically.
	jwtPath = "/home/pi/iotcore.jwt"
)

func certPath(keyPath string) string {
	ext := path.Ext(keyPath)
	return keyPath[:len(keyPath)-len(ext)] + certExtension
}

func parseDeviceConfig(filepath string) (iotcore.Device, error) {
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return iotcore.Device{}, err
	}

	var device iotcore.Device
	if err := json.Unmarshal(b, &device); err != nil {
		return iotcore.Device{}, err
	}

	if device.DeviceID == "" {
		deviceID, err := iotcore.DeviceIDFromCert(certPath(device.PrivKeyPath))
		if err != nil {
			return iotcore.Device{}, err
		}
		device.DeviceID = deviceID
	}

	return device, nil
}

func existingJWT(device iotcore.Device) (string, error) {
	f, err := os.Open(jwtPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}

		// There is no existing JWT.
		return "", fmt.Errorf("%s does not exist", jwtPath)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	jwt := string(b)
	if ok, err := device.VerifyJWT(jwt); !ok {
		return "", err
	}

	return jwt, nil
}

func newClient(device iotcore.Device) (mqtt.Client, error) {
	mqttOptions, err := iotcore.NewMQTTOptions(device, iotcore.DefaultBroker, caCerts)
	if err != nil {
		return nil, err
	}

	jwt, err := existingJWT(device)
	if err != nil {
		jwt, err = device.NewJWT(60 * time.Minute)
		if err != nil {
			return nil, err
		}

		// Persist the JWT.
		if err := ioutil.WriteFile(jwtPath, []byte(jwt), 0600); err != nil {
			log.Printf("Failed to save JWT to %s: %v", jwtPath, err)
		}
	}
	mqttOptions.SetPassword(jwt)

	return mqtt.NewClient(mqttOptions), nil
}
