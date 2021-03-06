package http

import (
	"errors"
	"fmt"
	"github.com/antonholmquist/jason"
	"k8-rolling-demo/lib/model"
	"net/http"
	"os"
	"sync"
)

type HttpClient struct {
	host      string
	port      string
	namespace string
}

func NewHttpClient(host, port, namespace string) *HttpClient {
	return &HttpClient{
		host:      host,
		port:      port,
		namespace: namespace,
	}
}

func DefaultHttpClient() *HttpClient {
	host := os.Getenv("K8_API_HOST")
	port := os.Getenv("K8_API_PORT")
	namespace := "k8-rolling-demo"

	return NewHttpClient(host, port, namespace)
}

func (c *HttpClient) FetchServiceDetails(service string) (*model.Service, error) {
	uri := fmt.Sprintf(
		"http://%s:%s/api/v1/namespaces/%s/services/%s",
		c.host,
		c.port,
		c.namespace,
		service,
	)

	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, errors.New(resp.Status)
	}

	v, err := jason.NewObjectFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	uid, _ := v.GetString("metadata", "uid")
	name, _ := v.GetString("metadata", "name")

	labels := make(map[string]string)
	labelsObject, _ := v.GetObject("metadata", "labels")
	for key, value := range labelsObject.Map() {
		str, _ := value.String()
		labels[key] = str
	}

	selectors := make(map[string]string)
	selectorsObject, _ := v.GetObject("spec", "selector")
	for key, value := range selectorsObject.Map() {
		str, _ := value.String()
		selectors[key] = str
	}

	svc := model.NewService(uid, name, labels, selectors)

	podNames, err := c.fetchPodNamesForService(name)
	if err != nil {
		return nil, err
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(podNames))

	for _, podName := range podNames {
		go func(podName string, wg *sync.WaitGroup) {
			pod, _ := c.FetchPodDetails(podName)
			svc.AddPod(pod)

			wg.Done()
		}(podName, wg)
	}

	wg.Wait()

	return svc, nil
}

func (c *HttpClient) fetchPodNamesForService(service string) ([]string, error) {
	uri := fmt.Sprintf(
		"http://%s:%s/api/v1/namespaces/%s/endpoints/%s",
		c.host,
		c.port,
		c.namespace,
		service,
	)

	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}

	v, err := jason.NewObjectFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	subsets, _ := v.GetObjectArray("subsets")
	if len(subsets) <= 0 {
		return nil, errors.New("No pods available in service")
	}

	addrs, _ := subsets[0].GetObjectArray("addresses")

	podNames := make([]string, 0)
	for _, addr := range addrs {
		podName, _ := addr.GetString("targetRef", "name")
		podNames = append(podNames, podName)
	}

	return podNames, nil
}

func (c *HttpClient) FetchPodDetails(pod string) (*model.Pod, error) {
	uri := fmt.Sprintf(
		"http://%s:%s/api/v1/namespaces/%s/pods/%s",
		c.host,
		c.port,
		c.namespace,
		pod,
	)

	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}

	v, err := jason.NewObjectFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	uid, _ := v.GetString("metadata", "uid")
	name, _ := v.GetString("metadata", "name")
	ipAddr, _ := v.GetString("status", "podIp")

	labels := make(map[string]string)
	labelsObject, _ := v.GetObject("metadata", "labels")
	for key, value := range labelsObject.Map() {
		str, _ := value.String()
		labels[key] = str
	}

	status, _ := v.GetString("status", "phase")
	conditions, _ := v.GetObjectArray("status", "conditions")
	firstCondition, _ := conditions[0].GetString("type")

	pd := &model.Pod{
		Uid:       uid,
		Name:      name,
		IpAddr:    ipAddr,
		Labels:    labels,
		Status:    status,
		Condition: firstCondition,
	}

	return pd, nil
}
