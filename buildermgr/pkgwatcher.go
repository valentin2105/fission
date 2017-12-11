/*
Copyright 2017 The Fission Authors.

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

package buildermgr

import (
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

type (
	packageWatcher struct {
		fissionClient    *crd.FissionClient
		builderNamespace string
		storageSvcUrl    string
	}
)

func makePackageWatcher(fissionClient *crd.FissionClient,
	builderNamespace string, storageSvcUrl string) *packageWatcher {
	pkgw := &packageWatcher{
		fissionClient:    fissionClient,
		builderNamespace: builderNamespace,
		storageSvcUrl:    storageSvcUrl,
	}
	return pkgw
}

func (pkgw *packageWatcher) build(pkg *crd.Package) {
	_, err := buildPackage(pkgw.fissionClient, pkgw.builderNamespace, pkgw.storageSvcUrl, pkg)
	if err != nil {
		log.Printf("Error building package %v: %v", pkg.Metadata.Name, err)
	}
}

func (pkgw *packageWatcher) watchPackages() {
	rv := ""
	for {
		wi, err := pkgw.fissionClient.Packages(metav1.NamespaceDefault).Watch(metav1.ListOptions{
			ResourceVersion: rv,
		})
		if err != nil {
			log.Fatalf("Error watching package CRD resources: %v", err)
		}

		for {
			ev, more := <-wi.ResultChan()
			if !more {
				break
			}
			if ev.Type == watch.Error {
				rv = ""
				time.Sleep(time.Second)
				break
			}
			pkg := ev.Object.(*crd.Package)
			rv = pkg.Metadata.ResourceVersion

			// only do build for packages in pending state
			if pkg.Status.BuildStatus == fission.BuildStatusPending {
				go pkgw.build(pkg)
			}
		}
	}
}
