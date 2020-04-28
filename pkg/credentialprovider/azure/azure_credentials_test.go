/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azure

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2019-05-01/containerregistry"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
)

type fakeClient struct {
	results []containerregistry.Registry
}

func (f *fakeClient) List(ctx context.Context) ([]containerregistry.Registry, error) {
	return f.results, nil
}

func Test(t *testing.T) {
	configStr := `
    {
        "aadClientId": "foo",
        "aadClientSecret": "bar"
    }`
	result := []containerregistry.Registry{
		{
			Name: to.StringPtr("foo"),
			RegistryProperties: &containerregistry.RegistryProperties{
				LoginServer: to.StringPtr("*.azurecr.io"),
			},
		},
		{
			Name: to.StringPtr("bar"),
			RegistryProperties: &containerregistry.RegistryProperties{
				LoginServer: to.StringPtr("*.azurecr.cn"),
			},
		},
		{
			Name: to.StringPtr("baz"),
			RegistryProperties: &containerregistry.RegistryProperties{
				LoginServer: to.StringPtr("*.azurecr.de"),
			},
		},
		{
			Name: to.StringPtr("bus"),
			RegistryProperties: &containerregistry.RegistryProperties{
				LoginServer: to.StringPtr("*.azurecr.us"),
			},
		},
	}
	fakeClient := &fakeClient{
		results: result,
	}

	provider := &acrProvider{
		registryClient: fakeClient,
	}
	provider.loadConfig(bytes.NewBufferString(configStr))

	creds := provider.Provide("")

	if len(creds) != len(result)+1 {
		t.Errorf("Unexpected list: %v, expected length %d", creds, len(result)+1)
	}
	for _, cred := range creds {
		if cred.Username != "" && cred.Username != "foo" {
			t.Errorf("expected 'foo' for username, saw: %v", cred.Username)
		}
		if cred.Password != "" && cred.Password != "bar" {
			t.Errorf("expected 'bar' for password, saw: %v", cred.Username)
		}
	}
	for _, val := range result {
		registryName := getLoginServer(val)
		if _, found := creds[registryName]; !found {
			t.Errorf("Missing expected registry: %s", registryName)
		}
	}
}

func TestCustomContainerRegistryUrls(t *testing.T) {
	azureStackFile, err := ioutil.TempFile("", "azurestack.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(azureStackFile.Name())

	azureStackConfig := []byte(`{"containerRegistryDNSSuffix": "azurecr.custom"}`)
	if _, err = azureStackFile.Write(azureStackConfig); err != nil {
		t.Fatal(err)
	}

	os.Setenv(azure.EnvironmentFilepathName, azureStackFile.Name())

	configStr := `
    {
        "cloud": "AzureStackCloud",
        "aadClientId": "foo",
        "aadClientSecret": "bar"
    }`
	result := []containerregistry.Registry{
		{
			Name: to.StringPtr("foo"),
			RegistryProperties: &containerregistry.RegistryProperties{
				LoginServer: to.StringPtr("*.azurecr.io"),
			},
		},
		{
			Name: to.StringPtr("bar"),
			RegistryProperties: &containerregistry.RegistryProperties{
				LoginServer: to.StringPtr("*.azurecr.cn"),
			},
		},
		{
			Name: to.StringPtr("baz"),
			RegistryProperties: &containerregistry.RegistryProperties{
				LoginServer: to.StringPtr("*.azurecr.de"),
			},
		},
		{
			Name: to.StringPtr("bus"),
			RegistryProperties: &containerregistry.RegistryProperties{
				LoginServer: to.StringPtr("*.azurecr.us"),
			},
		},
		{
			Name: to.StringPtr("custom"),
			RegistryProperties: &containerregistry.RegistryProperties{
				LoginServer: to.StringPtr("*.azurecr.custom"),
			},
		},
	}
	fakeClient := &fakeClient{
		results: result,
	}

	provider := &acrProvider{
		registryClient: fakeClient,
	}
	provider.loadConfig(bytes.NewBufferString(configStr))

	creds := provider.Provide("")

	if len(creds) != len(result)+1 {
		t.Errorf("Unexpected list: %v, expected length %d", creds, len(result)+1)
	}
	for _, cred := range creds {
		if cred.Username != "" && cred.Username != "foo" {
			t.Errorf("expected 'foo' for username, saw: %v", cred.Username)
		}
		if cred.Password != "" && cred.Password != "bar" {
			t.Errorf("expected 'bar' for password, saw: %v", cred.Username)
		}
	}
	for _, val := range result {
		registryName := getLoginServer(val)
		if _, found := creds[registryName]; !found {
			t.Errorf("Missing expected registry: %s", registryName)
		}
	}
}

func TestGetACRRE(t *testing.T) {
	tests := []struct {
		urls     []string
		expected string
	}{
		{
			urls:     []string{"*.azurecr.foo", "*.azurecr.bar"},
			expected: `.*\.azurecr\.foo|.*\.azurecr\.bar`,
		},
		{
			urls:     []string{"*.azurecr.foo.bar"},
			expected: `.*\.azurecr\.foo\.bar`,
		},
	}
	for _, test := range tests {
		if acrRE := getACRRE(test.urls); acrRE.String() != test.expected {
			t.Errorf("function makeACRRE returns \"%s\" for urls %+v, expected \"%s\"", acrRE, test.urls, test.expected)
		}
	}
}

func TestParseACRLoginServerFromImage(t *testing.T) {
	configStr := `
    {
        "aadClientId": "foo",
        "aadClientSecret": "bar"
    }`
	result := []containerregistry.Registry{
		{
			Name: to.StringPtr("foo"),
			RegistryProperties: &containerregistry.RegistryProperties{
				LoginServer: to.StringPtr("*.azurecr.io"),
			},
		},
	}
	fakeClient := &fakeClient{
		results: result,
	}

	provider := &acrProvider{
		registryClient: fakeClient,
	}
	provider.loadConfig(bytes.NewBufferString(configStr))
	provider.environment = &azure.Environment{
		ContainerRegistryDNSSuffix: ".azurecr.my.cloud",
	}
	tests := []struct {
		image    string
		expected string
	}{
		{
			image:    "invalidImage",
			expected: "",
		},
		{
			image:    "docker.io/library/busybox:latest",
			expected: "",
		},
		{
			image:    "foo.azurecr.io/bar/image:version",
			expected: "foo.azurecr.io",
		},
		{
			image:    "foo.azurecr.cn/bar/image:version",
			expected: "foo.azurecr.cn",
		},
		{
			image:    "foo.azurecr.de/bar/image:version",
			expected: "foo.azurecr.de",
		},
		{
			image:    "foo.azurecr.us/bar/image:version",
			expected: "foo.azurecr.us",
		},
		{
			image:    "foo.azurecr.my.cloud/bar/image:version",
			expected: "foo.azurecr.my.cloud",
		},
	}
	provider := &acrProvider{
		acrRE: getACRRE(defaultContainerRegistryUrls),
	}
	for _, test := range tests {
		if loginServer := provider.parseACRLoginServerFromImage(test.image); loginServer != test.expected {
			t.Errorf("function parseACRLoginServerFromImage returns \"%s\" for image %s, expected \"%s\"", loginServer, test.image, test.expected)
		}
	}
}
