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

package controller

import (
	"fmt"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"

	"github.com/GoogleCloudPlatform/gke-managed-certs/pkg/utils/types"
)

func (c *controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}

	glog.Infof("Enqueuing ManagedCertificate: %+v", obj)
	c.queue.AddRateLimited(key)
}

func (c *controller) enqueueAll() {
	mcrts, err := c.lister.List(labels.Everything())
	if err != nil {
		runtime.HandleError(err)
		return
	}

	if len(mcrts) <= 0 {
		glog.Info("No ManagedCertificates found in cluster")
		return
	}

	var names []string
	statuses := make(map[string]int, 0)
	for _, mcrt := range mcrts {
		names = append(names, mcrt.Name)
		statuses[mcrt.Status.CertificateStatus]++
	}

	for _, mcrt := range mcrts {
		c.enqueue(mcrt)
	}

	c.metrics.ObserveManagedCertificatesStatuses(statuses)
}

func (c *controller) handle(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	if err := c.sync.ManagedCertificate(types.NewCertId(namespace, name)); err != nil {
		return err
	}

	return err
}

func (c *controller) processNextManagedCertificate() {
	obj, shutdown := c.queue.Get()

	if shutdown {
		return
	}

	defer c.queue.Done(obj)

	key, ok := obj.(string)
	if !ok {
		c.queue.Forget(obj)
		runtime.HandleError(fmt.Errorf("Expected string in queue but got %T", obj))
		return
	}

	err := c.handle(key)
	if err == nil {
		c.queue.Forget(obj)
		return
	}

	c.queue.AddRateLimited(obj)
	runtime.HandleError(err)
}
