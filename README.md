# Zigflow: Declarative workflows for Temporal

[![Contributions Welcome](https://img.shields.io/badge/Contributions-Welcome-green.svg?style=flat)](https://github.com/mrsimonemms/zigflow/issues)
[![Licence](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/mrsimonemms/zigflow/blob/master/LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/mrsimonemms/zigflow?label=Release)](https://github.com/mrsimonemms/zigflow/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/mrsimonemms/zigflow)](https://goreportcard.com/report/github.com/mrsimonemms/zigflow)

Zigflow provides a **simple and declarative way** to define and manage
[Temporal](https://temporal.io) workflows using the
[CNCF Serverless Workflow](https://serverlessworkflow.io) specification. It
enables **low-code** and **no-code** workflow creation that's
**easy to visualize, share, and maintain**, without sacrificing the power and
reliability of Temporal.

---

## ✨ Features

* ✅ **CNCF Standard** – fully aligned with Serverless Workflow v1.0+
* ✅ **Low-code & Visual-ready** – ideal for UI workflow builders and orchestration
  tools
* ✅ **Powered by Temporal** – battle-tested reliability, retries, and state management
* ✅ **Kubernetes-native** – includes a Helm chart for easy deployment
* ✅ **Open & Extensible** – customize, extend, and contribute easily

---

## ⚡️ ZigFlow?

Ziggy is [Temporal's official mascot](https://temporal.io/blog/temporal-in-space).
They're a microscopic animal that is basically indestructible.

Sound familiar?

---

## 🧩 Example

Define a workflow declaratively in YAML:

```yaml
document:
  dsl: 1.0.0
  namespace: MoneyTransfer # Mapped to the task queue
  name: AccountTransferWorkflow # Workflow name
  version: 0.0.1
  title: Money Transfer Demo
  summary: Temporal's world-famous Money Transfer Demo, in DSL form
do:
  - queryState:
      listen:
        to:
          one:
            with:
              # ID maps to the query name in Temporal
              id: transferStatus
              # Temporal query - used to make read requests
              type: query
              # The data returned from the query - for application/json, this must be a string so Go interpolation works correctly
              data:
                approvalTime: ${ .data.stateApprovalTime }
                chargeResult:
                  chargeId: ${ .data.stateChargeId }
                progressPercentage: ${ .data.stateProgressPercentage }
                transferState: ${ .data.stateTransferState }
                workflowStatus: ${ .data.stateWorkflowStatus }
  - setup:
      set:
        idempotencyKey: ${ uuid }
        stateApprovalTime: 30
        stateChargeId: ${ uuid }
        stateProgressPercentage: 0
        stateTransferState: starting
        stateWorkflowStatus: ""
  - validate:
      call: http
      with:
        method: post
        endpoint: http://server:3000/validate
  - updateState:
      set:
        stateProgressPercentage: 25
        stateTransferState: running
  - withdraw:
      call: http
      with:
        method: post
        endpoint: http://server:3000/withdraw
        headers:
          content-type: application/json
        body:
          amount: ${ .input.amount }
          attempt: ${ .data.activity.attempt }
          idempotencyKey: ${ .data.idempotencyKey }
          name: ${ .data.workflow.workflow_type_name }
  - updateState:
      set:
        stateProgressPercentage: 50
  - deposit:
      call: http
      with:
        method: post
        endpoint: http://server:3000/deposit
        headers:
          content-type: application/json
        body:
            amount: ${ .input.amount }
            attempt: ${ .data.activity.attempt }
            idempotencyKey: ${ .data.idempotencyKey }
            name: ${ .data.workflow.workflow_type_name }
  - updateState:
      set:
        stateProgressPercentage: 75
  - sendNotification:
      call: http
      with:
        method: post
        endpoint: http://server:3000/notify
        headers:
          content-type: application/json
        body:
          amount: ${ .input.amount }
          fromAccount: ${ .input.fromAccount }
          toAccount: ${ .input.toAccount }
  - updateState:
      set:
        stateProgressPercentage: 100
        stateTransferState: finished
```

Run it through Zigflow:

```bash
zigflow -f ./path/to/workflow.yaml
```

This builds your Temporal workflow and runs the workers — no additional Go
boilerplate required.

You can now run it with any [Temporal SDK](https://docs.temporal.io/encyclopedia/temporal-sdks).

* [**Task Queue**](https://docs.temporal.io/task-queue): `zigflow`
* [**Workflow Type**](https://docs.temporal.io/workflows#intro-to-workflows):
  `example`

---

## 🧭 Related Projects

* [Temporal](https://temporal.io)
* [CNCF Serverless Workflow](https://serverlessworkflow.io)
* [Helm Chart Repository](./charts//zigflow)

---

## 🤝 Contributing

Contributions are welcome!

### Open in a container

* [Open in a container](https://code.visualstudio.com/docs/devcontainers/containers)

### Commit style

All commits must be done in the [Conventional Commit](https://www.conventionalcommits.org)
format.

```git
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

---

## ⭐️ Contributors

<a href="https://github.com/mrsimonemms/zigflow/graphs/contributors">
  <img alt="Contributors"
    src="https://contrib.rocks/image?repo=mrsimonemms/zigflow" />
</a>

Made with [contrib.rocks](https://contrib.rocks).

---

## 🪪 License

Distributed under the [Apache-2.0](./LICENSE) license

© 2025 [Zigflow authors](https://github.com/mrsimonemms/zigflow/graphs/contributors>)
