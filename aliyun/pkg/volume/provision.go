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
	"github.com/kubernetes-incubator/external-storage/lib/util"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	flexvolumeVendor = "aliyun"
	flexvolumeDriver = "disk"

	// are we allowed to set this? else make up our own
	annCreatedBy = "kubernetes.io/createdby"
	createdBy    = "aliyun-provisioner"

	annVolumeID = "aliyun.external-storage.incubator.kubernetes.io/VolumeID"
)

// NewAliyunProvisioner creates a new aliyun provisioner
func NewAliyunProvisioner(client kubernetes.Interface, ecsClient *ecs.Client) controller.Provisioner {
	provisioner := &aliyunProvisioner{
		client:    client,
		ecsClient: ecsClient,
	}

	return provisioner
}

type aliyunProvisioner struct {
	client    kubernetes.Interface
	ecsClient *ecs.Client
}

var _ controller.Provisioner = &aliyunProvisioner{}

// https://github.com/kubernetes-incubator/external-storage/blob/e26435c2ccd9ed5d2a60c838a902d22a3ec6ef5c/iscsi/targetd/provisioner/iscsi-provisioner.go#L102
// getAccessModes returns access modes aliyun Block Storage volume supports.
func (p *aliyunProvisioner) getAccessModes() []v1.PersistentVolumeAccessMode {
	return []v1.PersistentVolumeAccessMode{
		v1.ReadWriteOnce,
	}
}

// Provision creates a volume i.e. the storage asset and returns a PV object for
// the volume.
func (p *aliyunProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	if !util.AccessModesContainedInAll(p.getAccessModes(), options.PVC.Spec.AccessModes) {
		return nil, fmt.Errorf("Invalid Access Modes: %v, Supported Access Modes: %v", options.PVC.Spec.AccessModes, p.getAccessModes())
	}

	diskid, size, err := p.createVolume(options)
	if err != nil {
		return nil, err
	}

	annotations := make(map[string]string)
	annotations[annCreatedBy] = createdBy
	annotations[annVolumeID] = diskid

	labels := map[string]string{}
	for key, value := range options.Parameters {
		labels[key] = value
	}

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        diskid,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): resource.MustParse(fmt.Sprintf("%dGi", size)),
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				FlexVolume: &v1.FlexVolumeSource{
					Driver:   fmt.Sprintf("%s/%s", flexvolumeVendor, flexvolumeDriver),
					Options:  map[string]string{"volumeId": diskid},
					ReadOnly: false,
				},
			},
		},
	}

	return pv, nil
}

func (p *aliyunProvisioner) createVolume(volumeOptions controller.VolumeOptions) (string, int, error) {
	region := common.Region(volumeOptions.Parameters["region"])
	zone := volumeOptions.Parameters["zone"]
	category := ecs.DiskCategory(volumeOptions.Parameters["type"])

	volSize := volumeOptions.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
	volSizeBytes := volSize.Value()
	volszInt := util.RoundUpSize(volSizeBytes, util.GiB)
	if volszInt < 20 {
		return "", 0, fmt.Errorf("Volume size should greater than 20Gi")
	}
	size := int(volszInt)

	createRequest := &ecs.CreateDiskArgs{
		RegionId:     region,
		ZoneId:       zone,
		Size:         size,
		DiskName:     volumeOptions.PVC.Name,
		DiskCategory: category,
	}

	glog.V(2).Infof("Creating disk %+v", createRequest)
	diskid, err := p.ecsClient.CreateDisk(createRequest)
	if err != nil {
		glog.Errorf("Failed to create volume %s, error: %s", volumeOptions, err.Error())
		return "", 0, err
	}

	if err := p.ecsClient.WaitForDisk(region, diskid, ecs.DiskStatusAvailable, 60); err != nil {
		return "", 0, nil
	}

	return diskid, size, nil
}
