# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

REGISTRY?=quay.io/external_storage
IMAGE_PROVISONER=$(REGISTRY)/aliyun-provisioner

VERSION := v0.1.2

all build:
	CGO_ENABLED=0 GOOS=linux go build ./cmd/aliyun-provisioner
.PHONY: all build

container: build quick-container
.PHONY: container

quick-container:
	cp aliyun-provisioner deploy/docker/aliyun-provisioner
	docker build -t $(IMAGE_PROVISONER):$(VERSION) -f deploy/docker/Dockerfile.provisioner deploy/docker
.PHONY: quick-container

push: container
	docker push $(IMAGE_PROVISONER):$(VERSION)
.PHONY: push

clean:
	rm -f aliyun-provisioner
	rm -f deploy/docker/aliyun-provisioner
.PHONY: clean
