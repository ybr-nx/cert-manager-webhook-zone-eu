package main

import (
	"context"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/cmd"
	"io"
	"io/ioutil"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"net/http"
	"os"
	"strings"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {

	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	cmd.RunWebhookServer(GroupName,
		&zoneEuDNSProviderSolver{},
	)
}

type zoneEuDNSProviderSolver struct {
	client *kubernetes.Clientset
}

type zoneEuDNSProviderConfig struct {
	SecretRef string `json:"secretName"`
	ZoneName string  `json:"zoneName"`
	ApiUrl string	 `json:"apiUrl"`
}

func (c *zoneEuDNSProviderSolver) Name() string {
	return "zone-eu"
}

func (c *zoneEuDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	klog.V(6).Infof("call function Present: namespace=%s, zone=%s, fqdn=%s", ch.ResourceNamespace, ch.ResolvedZone, ch.ResolvedFQDN)

	config, err := clientConfig(c, ch)

	if err != nil {
		return fmt.Errorf("unable to get secret `%s`; %v", ch.ResourceNamespace, err)
	}

	addTxtRecord(config, ch)

	klog.Infof("Presented txt record %v", ch.ResolvedFQDN)

	return nil
}


func (c *zoneEuDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	config, err := clientConfig(c, ch)

	if err != nil {
		return fmt.Errorf("unable to get secret `%s`; %v", ch.ResourceNamespace, err)
	}

	var url = config.ApiUrl + "/dns/" + config.ZoneName + "/txt"

	// Get all DNS records
	dnsRecords, err := callDnsApi(url, "GET", nil, config)

	if err != nil {
		return fmt.Errorf("unable to get DNS records %v", err)
	}

	// Unmarshall response
	records := TxtRecordResponse{}
	readErr := json.Unmarshal(dnsRecords, &records)

	if readErr != nil {
		return fmt.Errorf("unable to unmarshal response %v", readErr)
	}

	var recordId string
	name := strings.TrimRight(ch.ResolvedFQDN, ".")
	for i := len(records.Records) - 1; i >= 0; i-- {
		if records.Records[i].Name == name {
			recordId = records.Records[i].Id
			break
		}
	}

	// Delete TXT record
	url = config.ApiUrl + "/dns/" + strings.TrimRight(config.ZoneName, ".") + "/txt/" + recordId
	del, err := callDnsApi(url, "DELETE", nil, config)

	if err != nil {
		klog.Error(err)
	}
	klog.Infof("TXT record deleted %s", string(del))
	return nil
}


func (c *zoneEuDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	k8sClient, err := kubernetes.NewForConfig(kubeClientConfig)
	klog.V(6).Infof("Input variable stopCh is %d length", len(stopCh))
	if err != nil {
		return err
	}

	c.client = k8sClient

	return nil
}

func loadConfig(cfgJSON *extapi.JSON) (zoneEuDNSProviderConfig, error) {
	cfg := zoneEuDNSProviderConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}


func stringFromSecretData(secretData *map[string][]byte, key string) (string, error) {
	data, ok := (*secretData)[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret data", key)
	}
	return string(data), nil
}

func addTxtRecord(config ApiConfig, ch *v1alpha1.ChallengeRequest) {
	url := config.ApiUrl + "/dns/" + strings.TrimRight(config.ZoneName, ".") + "/txt"

	name := strings.TrimRight(ch.ResolvedFQDN, ".")

	var jsonStr = fmt.Sprintf(`{"destination":"%s", "name":"%s"}`, ch.Key, name)

	add, err := callDnsApi(url, "POST", bytes.NewBuffer([]byte(jsonStr)), config)

	if err != nil {
		klog.Error(err)
	}

	klog.Infof("Added TXT record result: %s", string (add))
}

func clientConfig(c *zoneEuDNSProviderSolver, ch *v1alpha1.ChallengeRequest) (ApiConfig, error) {
	var config ApiConfig
	

	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return config, err
	}
	config.ZoneName = cfg.ZoneName
	config.ApiUrl = cfg.ApiUrl

	secretName := cfg.SecretRef

	sec, err := c.client.CoreV1().Secrets(ch.ResourceNamespace).Get(context.TODO(), secretName, metav1.GetOptions{})

	if err != nil {
		return config, fmt.Errorf("unable to get secret `%s/%s`; %v", secretName, ch.ResourceNamespace, err)
	}

	apiKey, err := stringFromSecretData(&sec.Data, "api-key")
	config.ApiKey = strings.TrimSpace(apiKey)

	if err != nil {
		return config, fmt.Errorf("unable to get api-key from secret `%s/%s`; %v", secretName, ch.ResourceNamespace, err)
	}

	apiUsername, err := stringFromSecretData(&sec.Data, "api-username")
	config.ApiUserName = strings.TrimSpace(apiUsername)

	if err != nil {
		return config, fmt.Errorf("unable to get api-username from secret `%s/%s`; %v", secretName, ch.ResourceNamespace, err)
	}

	return config, nil
}

func callDnsApi (url string, method string, body io.Reader, config ApiConfig) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return []byte{}, fmt.Errorf("unable to execute request %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Api-Compatibility", "2.1")
	req.SetBasicAuth(config.ApiUserName, config.ApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			klog.Fatal(err)
		}
	}()

	respBody, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent {
		return respBody, nil
	}

	text := "Error calling API status:" + resp.Status + " url: " +  url + " method: " + method
	klog.Error(text)
	return nil, errors.New(text)
}