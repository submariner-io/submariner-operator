package datafile

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"

	v1 "k8s.io/api/core/v1"
)

type SubctlData struct {
	BrokerURL   string     `json:"brokerURL"`
	ClientToken *v1.Secret `omitempty,json:"clientToken"`
	IPSecPSK    *v1.Secret `omitempty,json:"ipsecPSK"`
}

func (data *SubctlData) ToString() (string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(jsonBytes), nil
}

func NewFromString(str string) (*SubctlData, error) {
	data := &SubctlData{}
	bytes, err := base64.URLEncoding.DecodeString(str)
	if err != nil {
		return nil, err
	}
	return data, json.Unmarshal(bytes, data)
}

func (data *SubctlData) WriteToFile(filename string) error {

	dataStr, err := data.ToString()
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(filename, []byte(dataStr), 0644); err != nil {
		return err
	}

	return nil
}

func NewFromFile(filename string) (*SubctlData, error) {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return NewFromString(string(dat))
}
