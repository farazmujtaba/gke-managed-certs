/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package ssl provides operations for manipulating SslCertificate GCE resources.
package ssl

import (
	"golang.org/x/oauth2"
	compute "google.golang.org/api/compute/v0.beta"

	"github.com/GoogleCloudPlatform/gke-managed-certs/pkg/config"
	"github.com/GoogleCloudPlatform/gke-managed-certs/pkg/utils/http"
)

const (
	typeManaged = "MANAGED"
)

type Ssl interface {
	Create(name string, domains []string) error
	Delete(name string) error
	Exists(name string) (bool, error)
	Get(name string) (*compute.SslCertificate, error)
}

type sslImpl struct {
	service   *compute.SslCertificatesService
	projectID string
}

func New(config *config.Config) (Ssl, error) {
	client := oauth2.NewClient(oauth2.NoContext, config.Compute.TokenSource)
	client.Timeout = config.Compute.Timeout

	service, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	return &sslImpl{
		service:   service.SslCertificates,
		projectID: config.Compute.ProjectID,
	}, nil
}

// Create creates a new SslCertificate resource.
func (s sslImpl) Create(name string, domains []string) error {
	sslCertificate := &compute.SslCertificate{
		Managed: &compute.SslCertificateManagedSslCertificate{
			Domains: domains,
		},
		Name: name,
		Type: typeManaged,
	}

	_, err := s.service.Insert(s.projectID, sslCertificate).Do()
	return err
}

// Delete deletes an SslCertificate resource.
func (s sslImpl) Delete(name string) error {
	_, err := s.service.Delete(s.projectID, name).Do()
	return err
}

// Exists returns true if an SslCertificate exists, false if it is deleted. Error is not nil if an error has occurred.
func (s sslImpl) Exists(name string) (bool, error) {
	_, err := s.Get(name)
	if err == nil {
		return true, nil
	}

	if http.IsNotFound(err) {
		return false, nil
	}

	return false, err
}

// Get fetches an SslCertificate resource.
func (s sslImpl) Get(name string) (*compute.SslCertificate, error) {
	return s.service.Get(s.projectID, name).Do()
}
