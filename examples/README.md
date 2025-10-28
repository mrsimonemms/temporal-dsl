# Examples

A collection of examples

<!-- toc -->

* [Applications](#applications)
* [Running](#running)
  * [Running the worker](#running-the-worker)
  * [Starting the workflow](#starting-the-workflow)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Applications

<!-- apps-start -->

| Name | Description |
| --- | --- |
| [Basic Workflow](./basic) | An example of how to use Serverless Workflow to define Temporal Workflows |
| [TypeScript](./typescript) | The basic example, but in TypeScript |

<!-- apps-end -->

## Running

> These commands should be run from the root directory

The `NAME` variable should be set to the example you wish to run (eg, `basic`)

### Running the worker

```sh
make worker NAME=<example>
```

### Starting the workflow

```sh
make start NAME=<example>
```
