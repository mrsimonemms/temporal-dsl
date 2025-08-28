/*
 * Copyright 2025 Simon Emms <simon@simonemms.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Connection, Client } from '@temporalio/client';
import { nanoid } from 'nanoid';

async function bootstrap() {
  const connection = await Connection.connect({ address: 'localhost:7233' });

  const client = new Client({
    connection,
  });

  const handle = await client.workflow.start('basic-typescript', {
    taskQueue: 'temporal-dsl',
    workflowId: `basic-${nanoid()}`,
    args: [
      {
        userId: 3,
      },
    ],
  });

  console.log(`Started workflow: ${handle.workflowId}`);

  console.log(await handle.result());
}

bootstrap().catch((err) => {
  console.error(err);
  process.exit(1);
});
