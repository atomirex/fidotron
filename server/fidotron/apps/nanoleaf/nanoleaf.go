package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fidotron"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type NanoleafState struct {
	Hue int
	Saturation int
	Brightness int
}

func main() {
	fidotron.Run(&nanoleafApp{})
}

type AuthResponse struct {
	AuthToken string `json:"auth_token"`
}

type DeviceStateValue struct {
	Value bool `json:"value"`
}

type DeviceState struct {
	On *DeviceStateValue `json:"on"`
}

type DeviceIntValue struct {
	Value int `json:"value"`
}

type DeviceBrightness struct {
	Brightness *DeviceIntValue `json:"brightness"`
}

type DeviceHue struct {
	Hue *DeviceIntValue `json:"hue"`
}

type DeviceSaturation struct {
	Saturation *DeviceIntValue `json:"sat"`
}

type NanoleafDevice struct {
	address   string
	authToken string
}

func NewNanoleafDevice(address string) *NanoleafDevice {
	return &NanoleafDevice{address: address}
}

func (n *NanoleafDevice) Authorized() bool {
	return n.authToken != ""
}

func (n *NanoleafDevice) Set(active bool) error {
	state := &DeviceState{On: &DeviceStateValue{Value: active}}
	return doPut("http://"+n.address+"/api/v1/"+n.authToken+"/state", state)
}

func (n *NanoleafDevice) SetColor(hue int, saturation int, brightness int) error {
	h := &DeviceHue{Hue: &DeviceIntValue{Value: hue}}
	doPut("http://"+n.address+"/api/v1/"+n.authToken+"/state/hue", h)

	b := &DeviceBrightness{Brightness: &DeviceIntValue{Value: brightness}}
	doPut("http://"+n.address+"/api/v1/"+n.authToken+"/state/brightness", b)

	s := &DeviceSaturation{Saturation: &DeviceIntValue{Value: saturation}}
	doPut("http://"+n.address+"/api/v1/"+n.authToken+"/state/sat", s)

	return nil
}

func (n *NanoleafDevice) Authorize() error {
	auth := &AuthResponse{}
	resp, err := http.Post("http://"+n.address+"/api/v1/new", "", nil)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	err = json.Unmarshal(data, auth)

	if err != nil {
		return err
	}

	n.authToken = auth.AuthToken

	return nil
}

func (n *NanoleafDevice) Revoke() error {
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", "http://"+n.address+"/api/v1/"+n.authToken, nil)

	resp, err := client.Do(req)

	defer resp.Body.Close()

	if err != nil {
		return err
	}

	n.authToken = ""
	return nil
}

func doPut(url string, payload interface{}) error {
	client := &http.Client{}

	data, err := json.Marshal(payload)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	resp, err := client.Do(req)

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("Error while processing request " + resp.Status)
	}

	if err != nil {
		fmt.Println("Error with put request " + err.Error())
	}

	return nil
}


type nanoleafApp struct {
}

func (a *nanoleafApp) Prepare() {
	fmt.Println("Nanoleaf prepare")
}

func (a *nanoleafApp) Start() {
	fmt.Println("Nanoleaf start")

	c := fidotron.NewClient()

	// Wait for any nanoleaf on mdns
	c.Subscribe("sys/mdns/#", fidotron.BasicSubscriber(func(topic string, payload []byte) {
		data := make(map[string] interface{})
		err := json.Unmarshal(payload, &data)

		if err == nil {
			if name, exists := data["Name"]; exists {
				nameString, ok := name.(string)
				if ok && strings.HasSuffix(nameString, "._nanoleafapi._tcp.local.") {
					if addr, exists := data["AddrV4"]; exists {
						addrString, ok := addr.(string)
						if ok {
							fmt.Println("Nanoleaf at IP", addrString)
							
							// Provision any nanoleaf on mdns
							go func() {
								n := NewNanoleafDevice(fmt.Sprintf("%s:16021", addrString))

								for !n.Authorized() {
									n.Authorize()
									if !n.Authorized() {
										<- time.After(2 * time.Second)
									}
								}

								c.Send(fmt.Sprintf("apps/nanoleaf/%s", addrString), "")

								n.Set(true)

								c.Subscribe(fmt.Sprintf("nanoleaf/%s", addrString), fidotron.BasicSubscriber(func(topic string, payload []byte) {
									fmt.Println("Handling", string(payload))
									
									state := &NanoleafState{}
									err := json.Unmarshal(payload, state)

									if err == nil {
										n.SetColor(state.Hue, state.Saturation, state.Brightness)
									} else {
										fmt.Println("Error with state", err)
									}
								}))
							} ()
						}
					}
				}
			}
		}
	}))

	// TODO push mdns lookup requests for nanoleaf

	// TODO broadcast new nanoleaf available via fidotron bus

	// TODO show user interface for it on fidotron bus

	for {
		time.Sleep(5000)
	}
}
