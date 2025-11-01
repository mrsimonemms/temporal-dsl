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
| [Child Workflow](./child-workflows) | An example of how to declare and use a child workflow |
| [Competing Concurrent Tasks](./competing-concurrent-tasks) | Have two tasks competing and the first to finish wins |
| [Conditional Workflow](./conditionally-execute) | Execute tasks conditionally |
| [Money Transfer Demo](./money-transfer) | Temporal's world-famous Money Transfer Demo, in Serverless Workflow form |
| [Multiple Workflows](./multiple-workflows) | Configure multiple Temporal workflows |
| [Listener Workflow (Query)](./query) | Listen for Temporal query events |
| [Erroring Workflow](./raise) | Raise a bug |
| [Listener Workflow (Signal)](./signal) | Listen for Temporal signal events |
| [Switch Workflow](./switch) | Perform a switch statement |
| [TypeScript](./typescript) | The basic example, but in TypeScript |
| [Listener Workflow (Update)](./update) | Listen for Temporal update events |

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
