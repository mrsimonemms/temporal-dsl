# Copyright 2025 Simon Emms <simon@simonemms.com>
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

EXAMPLES = ./examples
TMP_IMG ?= ttl.sh/temporal-dsl
TMP_IMG_TAG ?= 24h

cruft-update:
ifeq (,$(wildcard .cruft.json))
	@echo "Cruft not configured"
else
	@cruft check || cruft update --skip-apply-ask --refresh-private-variables
endif
.PHONY: cruft-update

helm_img:
	@docker build -t ${TMP_IMG}:${TMP_IMG_TAG} .
	@docker push ${TMP_IMG}:${TMP_IMG_TAG}
.PHONY: helm_img

helm:
# Put your custom values in here
	@touch values.example.yaml
	@helm upgrade \
		--atomic \
		--cleanup-on-fail \
		--create-namespace \
		--install \
		--namespace dsl \
		--reset-values \
		--set image.pullPolicy=Always \
		--set image.repository=${TMP_IMG} \
		--set image.tag=${TMP_IMG_TAG} \
		--values ./values.example.yaml \
		--wait \
		dsl ./charts/temporal-dsl
.PHONY: helm

minikube:
	@minikube profile list | grep minikube | grep OK || minikube start
.PHONY: minikube

start:
	$(shell if [ -z "${NAME}" ]; then echo "NAME must be set"; exit 1; fi)
	go run ./examples/${NAME}
.PHONY: start

worker:
	$(shell if [ -z "${NAME}" ]; then echo "NAME must be set"; exit 1; fi)
	go run . -f ./examples/${NAME}/workflow.yaml
.PHONY: worker
