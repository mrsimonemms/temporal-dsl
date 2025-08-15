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

| Name | Description |
| --- | --- |
| [Basic](./basic/) | A basic application to show the concepts |
| [Conditionally Execute](./conditionally-execute/) | Allow tasks to only execute if they meet certain conditions |
| [Multiple Workflows](./multiple-workflows/) | Configure multiple workflows |
| [Listen](./listen/) | Configure listeners |
| [Money Transfer](./money-transfer/) | Temporal's world-famous Money Transfer Demo, in Serverless Workflow form - uses Docker Compose |
| [Query](./query/) | Configure query listener |
| [Raise](./raise/) | Raise an error |
| [Signal](./signal/) | Configure signal listener |

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
