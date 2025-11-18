# Copyright 2025 Temporal DSL authors <https://github.com/mrsimonemms/temporal-dsl/graphs/contributors>
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


"""Run a Temporal worker for the default task queue."""

from __future__ import annotations

import asyncio
import os
import sys
from typing import Any, Dict

from temporalio.client import Client
from temporalio.worker import Worker

from activities import greet
from workflows import SayHelloWorkflow


async def main() -> None:
    address = os.getenv("TEMPORAL_ADDRESS", "localhost:7233")
    namespace = os.getenv("TEMPORAL_NAMESPACE", "default")
    api_key = os.getenv("TEMPORAL_API_KEY")
    tls_enabled = os.getenv("TEMPORAL_TLS", "false").lower() == "true"

    connect_kwargs: Dict[str, Any] = {"namespace": namespace}
    if tls_enabled:
        connect_kwargs["tls"] = True
    if api_key:
        connect_kwargs["api_key"] = api_key

    client = await Client.connect(address, **connect_kwargs)

    task_queue = os.getenv("PYTHON_CHILD_TASK_QUEUE", "python-child")

    worker = Worker(
        client=client,
        task_queue=task_queue,
        workflows=[SayHelloWorkflow],
        activities=[greet],
    )

    print(f"Worker listening on task queue: {task_queue}")

    await worker.run()


def _run() -> None:
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("Worker shut down", file=sys.stderr)
    except Exception as exc:  # pragma: no cover - CLI surface
        print(exc, file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    _run()
