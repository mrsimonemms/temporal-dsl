# Temporal DSL

<!-- markdownlint-disable-next-line MD013 MD034 -->
[![Go Report Card](https://goreportcard.com/badge/github.com/mrsimonemms/temporal-dsl)](https://goreportcard.com/report/github.com/mrsimonemms/temporal-dsl)

Build [Temporal](https://temporal.io) workflows from YAML

<!-- toc -->

* [Loom](#loom)
* [Why?](#why)
* [Architecture](#architecture)
  * [Goals](#goals)
* [Getting started](#getting-started)
  * [Define your workflow](#define-your-workflow)
  * [Start your Temporal server](#start-your-temporal-server)
  * [Run](#run)
    * [Running examples](#running-examples)
* [Schema](#schema)
  * [Variables](#variables)
* [Future developments](#future-developments)
  * [Implementation roadmap](#implementation-roadmap)
* [Contributing](#contributing)
  * [Open in a container](#open-in-a-container)
  * [Commit style](#commit-style)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Loom

Here's a Loom video explaining the idea

[![Video thumbnail](https://cdn.loom.com/sessions/thumbnails/ebd04b2eeda645bab5ad4426ba6f636a-18b80f0119063d98-full-play.gif)](https://www.loom.com/share/ebd04b2eeda645bab5ad4426ba6f636a)

## Why?

We regularly get asked about using a domain-specific language (DSL) to make Temporal
workflows in a no/low-code format. For very good reasons, this is outside the scope
of the Temporal project. [But still they come](https://www.youtube.com/watch?v=Poii8JAbtng&t=441s).

[Serverless Workflow](https://serverlessworkflow.io) is a [CNCF](https://www.cncf.io/)-sponsored
project to create a vendor-neutral method of defining a workflow.

## Architecture

This application is designed to replace the Temporal worker by generating the
workflows and activities defined. This will be a long-running application.

Triggering workflows defined by this application will be done in the normal
Temporal way - see [examples](./examples) for more information.

Conceptually, the items in the `do` array of the schema are things that a Temporal
workflow can do. Sometimes these will be invoked by the [workflow](https://docs.temporal.io/workflows),
but most will be [activities](https://docs.temporal.io/activities).

### Goals

This project has a number of goals:
1. _Can_ a Temporal workflow be configured by a DSL?
1. _Should_ a Temporal workflow be configured by a DSL?
1. Would this be a useful addition to the Temporal ecosystem?

From a deployment point of view, this is designed as a standalone application.
This allows for it to be run in multiple different ways, such as in Kubernetes
or on bare metal.

## Getting started

### Define your workflow

Create a workflow in the [Serverless Workflow](https://serverlessworkflow.io)
schema.

An [example workflow is provided](./workflow.example.yaml).

### Start your Temporal server

This can be any flavour of Temporal (Cloud or self-hosted). To start a local
development server, run:

```sh
temporal server start-dev
```

### Run

There are many options available so it's best to look at the available options
in the application's `help`.

```sh
go run . -h
```

At it's simplest, this will be:

```sh
go run . --temporal-address localhost:7233 --files ./workflow.example.yaml
```

It's now ready for all your workflow needs

#### Running examples

See [examples](./examples) directory

## Schema

### Variables

Each call receives the input and output from previous calls, so that can be
used in calls. This is done using the [Go template](https://pkg.go.dev/text/template)
variable methods - if you set an `userId` variable, this can be retrieved by
adding `{{ .userId }}` in your schema definition.

Environment variables are also used, provided that the match the prefix - by default,
this is `TDSL_`. These can also be parsed - the variable `TDSL_EXAMPLE_ENVVAR`
would be retrieved by adding `{{ .TDSL_EXAMPLE_ENVVAR }}` to your schema definition.

## Future developments

This is largely dependent upon how much interest there in the community, so please
add stars and [raise pull requests](#contributing).

This is a very basic implementation, with only support for the basic Temporal
workflow. Things that will be needed to make this a production-grade system (not
exhaustive):
* [Temporal worker versioning](https://docs.temporal.io/production-deployment/worker-deployments/worker-versioning)
* Custom tasks, especially to call user-defined activities
* Published release/Helm chart

### Implementation roadmap

The table below lists the current state of this implementation. This table is a
roadmap for the project based on the
[DSL Reference doc](https://github.com/serverlessworkflow/specification/blob/v1.0.0/dsl-reference.md).

This project currently only supports DSL `1.0.0`.

## Contributing

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
