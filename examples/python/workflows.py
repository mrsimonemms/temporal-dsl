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


"""Workflows for the Python Temporal example."""

from __future__ import annotations

from datetime import timedelta
from typing import Any, Mapping, Optional

from temporalio import workflow

from activities import greet


@workflow.defn(name="say-hello-workflow")
class SayHelloWorkflow:
    """Workflow that greets the supplied name via an activity."""

    @workflow.run
    async def run(
        self,
        initial_input: Optional[Mapping[str, Any]] = None,
        state_payload: Optional[Mapping[str, Any]] = None,
    ) -> str:
        _ = state_payload  # currently unused but retained for parity with DSL signature
        # The DSL parent workflow passes the original input as the first argument.
        # Fall back to a reasonable default if the expected key is missing.
        name = "Temporal"
        if isinstance(initial_input, Mapping):
            candidate = initial_input.get("name")
            if candidate is not None:
                name = str(candidate)

        return await workflow.execute_activity(
            greet,
            name,
            schedule_to_close_timeout=timedelta(seconds=10),
        )
