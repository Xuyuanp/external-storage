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

package volume

import (
	"fmt"

	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"k8s.io/api/core/v1"
)

func (p *aliyunProvisioner) Delete(volume *v1.PersistentVolume) error {
	glog.Infof("Delete called for volume: %s", volume.Name)

	if !p.provisioned(volume) {
		strerr := fmt.Sprintf("this provisioner %s didn't provision volume %q and so can't delete it", createdBy, volume.Name)
		return &controller.IgnoredError{Reason: strerr}
	}

	if region, ok := volume.Labels["region"]; ok {
		if err := p.ecsClient.WaitForDisk(common.Region(region), volume.Name, ecs.DiskStatusAvailable, 30); err != nil {
			glog.Warningf("Volume %s is not in Available status", volume)
			return err
		}
	} else {
		glog.Warningf("Volume %s has no 'region' field in labels", volume)
	}

	if err := p.ecsClient.DeleteDisk(volume.Name); err != nil {
		glog.Errorf("Failed to delete volume %s, error: %s", volume, err.Error())
		return err
	}
	return nil
}

func (p *aliyunProvisioner) provisioned(volume *v1.PersistentVolume) bool {
	creator, ok := volume.Annotations[annCreatedBy]
	return ok && creator == createdBy
}
