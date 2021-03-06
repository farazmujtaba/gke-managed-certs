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

// Package config manages configuration of the whole application.
package config

import (
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/golang/glog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v0.beta"
	gcfg "gopkg.in/gcfg.v1"
	"k8s.io/kubernetes/pkg/cloudprovider/providers/gce"
)

const (
	managedActive                        = "Active"
	managedEmpty                         = ""
	managedFailedCaaChecking             = "FailedCaaChecking"
	managedFailedCaaForbidden            = "FailedCaaForbidden"
	managedFailedNotVisible              = "FailedNotVisible"
	managedFailedRateLimited             = "FailedRateLimited"
	managedProvisioning                  = "Provisioning"
	managedProvisioningFailed            = "ProvisioningFailed"
	managedProvisioningFailedPermanently = "ProvisioningFailedPermanently"
	managedRenewalFailed                 = "RenewalFailed"

	SslCertificateNamePrefix = "mcrt-"

	sslActive                              = "ACTIVE"
	sslEmpty                               = ""
	sslFailedCaaChecking                   = "FAILED_CAA_CHECKING"
	sslFailedCaaForbidden                  = "FAILED_CAA_FORBIDDEN"
	sslFailedNotVisible                    = "FAILED_NOT_VISIBLE"
	sslFailedRateLimited                   = "FAILED_RATE_LIMITED"
	sslManagedCertificateStatusUnspecified = "MANAGED_CERTIFICATE_STATUS_UNSPECIFIED"
	sslProvisioning                        = "PROVISIONING"
	sslProvisioningFailed                  = "PROVISIONING_FAILED"
	sslProvisioningFailedPermanently       = "PROVISIONING_FAILED_PERMANENTLY"
	sslRenewalFailed                       = "RENEWAL_FAILED"
)

type computeConfig struct {
	TokenSource oauth2.TokenSource
	ProjectID   string
	Timeout     time.Duration
}

type certificateStatusConfig struct {
	// Certificate is a mapping from SslCertificate status to ManagedCertificate status
	Certificate map[string]string
	// Domain is a mapping from SslCertificate domain status to ManagedCertificate domain status
	Domain map[string]string
}

type Config struct {
	// CertificateStatus holds mappings of SslCertificate statuses to ManagedCertificate statuses
	CertificateStatus certificateStatusConfig
	// Compute is GCP-specific configuration
	Compute computeConfig
	// SslCertificateNamePrefix is a prefix prepended to SslCertificate resources created by the controller
	SslCertificateNamePrefix string
}

func New(gceConfigFilePath string) (*Config, error) {
	tokenSource, projectID, err := getTokenSourceAndProjectID(gceConfigFilePath)
	if err != nil {
		return nil, err
	}

	glog.Infof("TokenSource: %#v, projectID: %s", tokenSource, projectID)

	domainStatuses := make(map[string]string, 0)
	domainStatuses[sslActive] = managedActive
	domainStatuses[sslFailedCaaChecking] = managedFailedCaaChecking
	domainStatuses[sslFailedCaaForbidden] = managedFailedCaaForbidden
	domainStatuses[sslFailedNotVisible] = managedFailedNotVisible
	domainStatuses[sslFailedRateLimited] = managedFailedRateLimited
	domainStatuses[sslProvisioning] = managedProvisioning

	certificateStatuses := make(map[string]string, 0)
	certificateStatuses[sslActive] = managedActive
	certificateStatuses[sslEmpty] = managedEmpty
	certificateStatuses[sslManagedCertificateStatusUnspecified] = managedEmpty
	certificateStatuses[sslProvisioning] = managedProvisioning
	certificateStatuses[sslProvisioningFailed] = managedProvisioningFailed
	certificateStatuses[sslProvisioningFailedPermanently] = managedProvisioningFailedPermanently
	certificateStatuses[sslRenewalFailed] = managedRenewalFailed

	return &Config{
		CertificateStatus: certificateStatusConfig{
			Certificate: certificateStatuses,
			Domain:      domainStatuses,
		},
		Compute: computeConfig{
			TokenSource: tokenSource,
			ProjectID:   projectID,
			Timeout:     30 * time.Second,
		},
		SslCertificateNamePrefix: SslCertificateNamePrefix,
	}, nil
}

func getTokenSourceAndProjectID(gceConfigFilePath string) (oauth2.TokenSource, string, error) {
	if gceConfigFilePath != "" {
		glog.V(1).Info("In a GKE cluster")

		config, err := os.Open(gceConfigFilePath)
		if err != nil {
			return nil, "", fmt.Errorf("Could not open cloud provider configuration %s: %v", gceConfigFilePath, err)
		}
		defer config.Close()

		var cfg gce.ConfigFile
		if err := gcfg.ReadInto(&cfg, config); err != nil {
			return nil, "", fmt.Errorf("Could not read config %v", err)
		}
		glog.Infof("Using GCE provider config %+v", cfg)

		return gce.NewAltTokenSource(cfg.Global.TokenURL, cfg.Global.TokenBody), cfg.Global.ProjectID, nil
	}

	projectID, err := metadata.ProjectID()
	if err != nil {
		return nil, "", fmt.Errorf("Could not fetch project id: %v", err)
	}

	if len(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")) > 0 {
		glog.V(1).Info("In a GCP cluster")
		tokenSource, err := google.DefaultTokenSource(oauth2.NoContext, compute.ComputeScope)
		return tokenSource, projectID, err
	} else {
		glog.V(1).Info("Using default TokenSource")
		return google.ComputeTokenSource(""), projectID, nil
	}
}
