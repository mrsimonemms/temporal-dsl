# Python

The [basic example](../basic/), but in Python

<!-- toc -->

* [Getting started](#getting-started)
* [Running the worker](#running-the-worker)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Getting started

```sh
uv sync
uv run python main.py
```

`uv sync` creates a local virtual environment (by default at `.venv/`) and
installs the dependencies declared in `pyproject.toml`. `uv run python main.py`
runs the script inside that environment, triggering the workflow with some
input data and printing everything to the console.

## Running the worker

```sh
uv run python worker.py
```

This launches a Temporal worker on the `python-child` task queue (override with
`PYTHON_CHILD_TASK_QUEUE`) with the `SayHelloWorkflow` child workflow and
`greet` activity defined in `workflows.py` and `activities.py`. The DSL
workflow uses the child workflow `namespace` field in `workflow.yaml` to select
that task queue, so leave this worker running while you start the parent
workflow.
