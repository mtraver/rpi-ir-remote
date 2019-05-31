package main

import (
	"encoding/json"
	"io/ioutil"
	"path"

	"github.com/mtraver/iotcore"
)

const (
	certExtension = ".x509"
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
