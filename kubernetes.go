// Based on https://github.com/kelseyhightower/konfd/blob/master/kubernetes.go
// which was
// Copyright 2016 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
//
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// ErrNotExist is an error returned when an object does not exist.
var ErrNotExist = errors.New("object does not exist")

// ObjectReference contains enough information to let you inspect or modify the referred object.
type ObjectReference struct {
	Kind            string `json:"kind,omitempty"`
	Namespace       string `json:"namespace,omitempty"`
	Name            string `json:"name,omitempty"`
	UID             string `json:"uid,omitempty"`
	APIVersion      string `json:"apiVersion,omitempty"`
	ResourceVersion string `json:"resourceVersion,omitempty"`
	FieldPath       string `json:"fieldPath,omitempty"`
}

// EventSource contains information for an event.
type EventSource struct {
	Component string `json:"component,omitempty"`
	Host      string `json:"host,omitempty"`
}

// Event is a report of an event somewhere in the cluster.
type Event struct {
	APIVersion     string           `json:"apiVersion"`
	Kind           string           `json:"kind"`
	Metadata       Metadata         `json:"metadata"`
	Count          int              `json:"count"`
	Message        string           `json:"message,omitempty"`
	Reason         string           `json:"reason,omitempty"`
	Source         EventSource      `json:"source"`
	Type           string           `json:"type"`
	FirstTimestamp time.Time        `json:"firstTimestamp"`
	LastTimestamp  time.Time        `json:"lastTimestamp"`
	InvolvedObject *ObjectReference `json:"involvedObject"`
}

type ConfigMapList struct {
	Items []ConfigMap `json:"items"`
}

type ConfigMap struct {
	ApiVersion string            `json:"apiVersion"`
	Data       map[string]string `json:"data"`
	Kind       string            `json:"kind"`
	Metadata   Metadata          `json:"metadata"`
}

type Metadata struct {
	Name            string            `json:"name"`
	GenerateName    string            `json:"generateName,omitempty"`
	Namespace       string            `json:"namespace"`
	Labels          map[string]string `json:"labels"`
	Annotations     map[string]string `json:"annotations"`
	ResourceVersion string            `json:"resourceVersion"`
	UID             string            `json:"uid"`
}

type k8sClient struct {
	endpoint string
	client   *http.Client
}

func newk8sClient(endpoint string) *k8sClient {
	if endpoint == "" {
		endpoint = "http://127.0.0.1:8001"
	}
	return &k8sClient{
		endpoint: endpoint,
		client:   &http.Client{},
	}
}

func (k *k8sClient) getConfigMaps(namespace, selector string) (*ConfigMapList, error) {
	path := "/api/v1/configmaps"
	if namespace != "" {
		path = "/api/v1/namespaces/" + namespace + "/configmaps"
	}
	if selector != "" {
		path = path + "?labelSelector=" + selector
	}

	resp, err := k.client.Get(k.endpoint + path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("non 200 response code")
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	var cl ConfigMapList
	err = json.Unmarshal(data, &cl)
	if err != nil {
		return nil, err
	}
	return &cl, nil
}

func newConfigMap(namespace, name string) *ConfigMap {
	c := &ConfigMap{
		ApiVersion: "v1",
		Data:       make(map[string]string),
		Kind:       "ConfigMap",
		Metadata: Metadata{
			Name:        name,
			Namespace:   namespace,
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
		},
	}
	return c
}

func (k *k8sClient) getConfigMap(namespace, name string) (*ConfigMap, error) {
	u := fmt.Sprintf("%s/api/v1/namespaces/%s/configmaps/%s", k.endpoint, namespace, name)
	resp, err := k.client.Get(u)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 404 {
		return nil, ErrNotExist
	}

	if resp.StatusCode != 200 {
		return nil, errors.New("non 200 response code")
	}
	defer resp.Body.Close()

	return configMapFromReader(resp.Body)
}

func (k *k8sClient) createConfigMap(c *ConfigMap) (*ConfigMap, error) {
	body, err := json.MarshalIndent(&c, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error encoding configmap %s: %v", c.Metadata.Name, err)
	}
	u := fmt.Sprintf("%s/api/v1/namespaces/%s/configmaps", k.endpoint, c.Metadata.Namespace)
	resp, err := http.Post(u, "", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creating configmap %s: %v", c.Metadata.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return nil, fmt.Errorf("error creating configmap %s; got HTTP %v status code", c.Metadata.Name, resp.StatusCode)
	}

	return configMapFromReader(resp.Body)
}

func (k *k8sClient) updateConfigMap(c *ConfigMap) (*ConfigMap, error) {
	body, err := json.MarshalIndent(&c, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error encoding configmap %s: %v", c.Metadata.Name, err)
	}

	u := fmt.Sprintf("%s/api/v1/namespaces/%s/configmaps/%s", k.endpoint, c.Metadata.Namespace, c.Metadata.Name)
	request, err := http.NewRequest(http.MethodPut, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error updating configmap %s: %v", c.Metadata.Name, err)
	}

	resp, err := k.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error updating configmap %s: %v", c.Metadata.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error updating configmap %s; got HTTP %v status code", c.Metadata.Name, resp.StatusCode)
	}

	return configMapFromReader(resp.Body)
}

func configMapFromReader(body io.Reader) (*ConfigMap, error) {
	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read")
	}

	var cm ConfigMap
	if err := json.Unmarshal(data, &cm); err != nil {
		return nil, errors.Wrap(err, "failed to unmarhsal")
	}
	return &cm, nil
}

func (k *k8sClient) waitForKubernetes() error {
	timeout := time.After(time.Minute)
	tick := time.Tick(5 * time.Second)
	for {
		select {
		case <-timeout:
			return errors.New("timed out waiting for Kubernetes")
		case <-tick:
			resp, err := http.Get(k.endpoint + "/api")
			if err == nil {
				resp.Body.Close()
				return nil
			}
		}
	}
}

func newEvent(namespace, name string) *Event {
	return &Event{
		APIVersion: "v1",
		Metadata: Metadata{
			Namespace:    namespace,
			GenerateName: name,
		},
	}
}

func (k *k8sClient) postEvent(e *Event) error {
	body, err := json.MarshalIndent(&e, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding event %s: %v", e.Metadata.GenerateName, err)
	}

	fmt.Println(string(body))

	u := fmt.Sprintf("%s/api/v1/namespaces/%s/events", k.endpoint, e.Metadata.Namespace)
	request, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error creating event %s: %v", e.Metadata.GenerateName, err)
	}

	resp, err := k.client.Do(request)
	if err != nil {
		return fmt.Errorf("error creating event %s: %v", e.Metadata.GenerateName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("error creating event %s; got HTTP %v status code", e.Metadata.GenerateName, resp.StatusCode)
	}

	return nil
}
