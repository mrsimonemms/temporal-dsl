# Temporal DSL: Declarative workflows for Temporal

[![Contributions Welcome](https://img.shields.io/badge/Contributions-Welcome-green.svg?style=flat)](https://github.com/mrsimonemms/temporal-dsl/issues)
[![Licence](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/mrsimonemms/temporal-dsl/blob/master/LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/mrsimonemms/temporal-dsl?label=Release)](https://github.com/mrsimonemms/temporal-dsl/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/mrsimonemms/temporal-dsl)](https://goreportcard.com/report/github.com/mrsimonemms/temporal-dsl)

Temporal DSL provides a **simple and declarative way** to define and manage
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

## 🧩 Example

Define a workflow declaratively in YAML:

```yaml
document:
  dsl: 1.0.0
  namespace: temporal-dsl # Mapped to the task queue
  name: example # Workflow name
  version: 0.0.1
  title: Example Workflow
  summary: An example of how to use Serverless Workflow to define Temporal Workflows
timeout:
  after:
    minutes: 1
# Validate the input schema
input:
  schema:
    format: json
    document:
      type: object
      required:
        - userId
      properties:
        userId:
          type: number
do:
  # Set some data to the state
  - step1:
      set:
        # Set a variable from an envvar
        envvar: ${ .env.EXAMPLE_ENVVAR }
        # Generate a UUID at a workflow level
        uuid: ${ uuid }
  # Pause the workflow
  - wait:
      wait:
        seconds: 5
  # Make an HTTP call, using the userId received from the input
  - getUser:
      call: http
      # Expose the response to the output
      export:
        as: user
      with:
        method: get
        endpoint: ${ "https://jsonplaceholder.typicode.com/users/" + (.input.userId | tostring) }
```

Run it through Temporal DSL:

```bash
temporal-dsl -f ./path/to/workflow.yaml
```

This builds your Temporal workflow and runs the workers — no additional Go
boilerplate required.

You can now run it with any [Temporal SDK](https://docs.temporal.io/encyclopedia/temporal-sdks).

* [**Task Queue**](https://docs.temporal.io/task-queue): `temporal-dsl`
* [**Workflow Type**](https://docs.temporal.io/workflows#intro-to-workflows):
  `example`

---

## 🧭 Related Projects

* [Temporal](https://temporal.io)
* [CNCF Serverless Workflow](https://serverlessworkflow.io)
* [Helm Chart Repository](./charts//temporal-dsl)

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

<a href="https://github.com/mrsimonemms/temporal-dsl/graphs/contributors">
  <img alt="Contributors"
    src="https://contrib.rocks/image?repo=mrsimonemms/temporal-dsl" />
</a>

Made with [contrib.rocks](https://contrib.rocks).

---

## 🪪 License

[Apache-2.0](./LICENSE) © [Temporal DSL authors](https://github.com/mrsimonemms/temporal-dsl/graphs/contributors>)
