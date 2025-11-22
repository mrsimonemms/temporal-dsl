# Copyright 2025 Zigflow authors <https://github.com/mrsimonemms/zigflow/graphs/contributors>
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

FROM golang AS builder
ARG GIT_COMMIT
ARG GIT_REPO="github.com/mrsimonemms/zigflow"
ARG PROJECT_NAME="zigflow"
ARG VERSION
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOCACHE=/go/.cache
ENV PROJECT_NAME="${PROJECT_NAME}"
USER 1000
WORKDIR /go/app
COPY --chown=1000:1000 . .
RUN go build \
  -ldflags \
  "-w -s -X $GIT_REPO/cmd.Version=$VERSION -X $GIT_REPO/cmd.GitCommit=$GIT_COMMIT" \
  -o /go/bin/app
COPY --from=cosmtrek/air /go/bin/air /go/bin/air
ENTRYPOINT [ "air" ]

FROM scratch
ARG GIT_COMMIT
ARG VERSION
ENV GIT_COMMIT="${GIT_COMMIT}"
ENV VERSION="${VERSION}"
ENV WORKFLOW_FILE=/workflow.yaml
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/bin/app /app
ENTRYPOINT [ "/app/app" ]
