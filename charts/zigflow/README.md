# zigflow

![Version: 0.0.0](https://img.shields.io/badge/Version-0.0.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)

Turn your declarative YAML into production-ready Temporal workflows

**Homepage:** <https://zigflow.dev>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Simon Emms | <simon@simonemms.com> | <https://simonemms.com> |

## Source Code

* <https://github.com/mrsimonemms/zigflow>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Node affinity |
| autoscaling.enabled | bool | `false` | Autoscaling enabled |
| autoscaling.maxReplicas | int | `100` | Maximum replicas |
| autoscaling.minReplicas | int | `1` | Minimum replicas |
| autoscaling.targetCPUUtilizationPercentage | int | `80` | When to trigger a new replica |
| config | object | `{"log-level":"info"}` | Accepts any of the command line arguments |
| envvars | string | `nil` | Additional environment variables |
| fullnameOverride | string | `""` | String to fully override names |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy |
| image.repository | string | `"ghcr.io/mrsimonemms/zigflow"` | Image repositiory |
| image.tag | string | `""` | Image tag - defaults to the chart's `Version` if not set |
| imagePullSecrets | list | `[]` | Docker registry secret names |
| livenessProbe.httpGet.path | string | `"/health"` | Path to demonstrate app liveness |
| livenessProbe.httpGet.port | string | `"health"` | Port to demonstrate app liveness |
| nameOverride | string | `""` | String to partially override name |
| nodeSelector | object | `{}` | Node selector |
| podAnnotations | object | `{}` | Pod [annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) |
| podLabels | object | `{}` | Pod [labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) |
| podSecurityContext | object | `{}` | Pod's [security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context) |
| readinessProbe.httpGet.path | string | `"/health"` | Path to demonstrate app readiness |
| readinessProbe.httpGet.port | string | `"health"` | Port to demonstrate app readiness |
| replicaCount | int | `1` | Number of replicas |
| resources | object | `{}` | Configure resources available |
| securityContext | object | `{}` | Container's security context |
| service.health.port | int | `3000` | Health service port |
| service.metrics.port | int | `9090` | Metrics service port |
| service.type | string | `"ClusterIP"` | Service type |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account |
| serviceAccount.automount | bool | `true` | Automatically mount a ServiceAccount's API credentials? |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and create is true, a name is generated using the fullname template |
| tolerations | list | `[]` | Node toleration |
| volumeMounts | list | `[]` | Additional volumeMounts on the output Deployment definition. |
| volumes | list | `[]` | Additional volumes on the output Deployment definition. |
| workflow.file | string | `"/data/workflow.yaml"` | Location the workflow volumes is mapped |
| workflow.inline | object | `{"do":[{"baseData":{"export":{"as":"data"},"set":{"array":["${ uuid }",{"hello":"world"}],"envvar":"${ .env.EXAMPLE_ENVVAR }","inputUserId":"${ .input.userId }","object":{"hello":"world","uuid":"${ uuid }"},"uuid":"${ uuid }"}}},{"wait":{"wait":{"seconds":5}}},{"getUser":{"call":"http","export":{"as":"getUser"},"with":{"endpoint":"${ \"https://jsonplaceholder.typicode.com/users/\" + (.input.userId | tostring) }","method":"get"}}},{"raiseAlarm":{"export":{"as":"raiseAlarm"},"fork":{"branches":[{"callNurse":{"call":"http","export":{"as":"callNurse"},"with":{"endpoint":"https://jsonplaceholder.typicode.com/users/2","method":"get"}}},{"multiStep":{"do":[{"wait1":{"wait":{"seconds":3}}},{"wait2":{"wait":{"seconds":2}}}]}},{"callDoctor":{"call":"http","export":{"as":"callDoctor"},"with":{"endpoint":"${ \"https://jsonplaceholder.typicode.com/users/\" + (.input.userId | tostring) }","method":"get"}}}],"compete":false}}}],"document":{"dsl":"1.0.0","name":"basic","namespace":"zigflow","summary":"An example of how to use Serverless Workflow to define Temporal Workflows","title":"Basic Workflow","version":"0.0.1"},"input":{"schema":{"document":{"properties":{"userId":{"type":"number"}},"required":["userId"],"type":"object"},"format":"json"}},"timeout":{"after":{"minutes":1}}}` | Workflow YAML |
| workflow.secret | string | `"workflow"` | Name of the secret containing `workflow.yaml` |
| workflow.useInline | bool | `true` | Use the inline workflow. If false, you must declare a secret with the workflow in `workflow.yaml` |

