<!--
  ~ Copyright 2025 Simon Emms <simon@simonemms.com>
  ~
  ~ Licensed under the Apache License, Version 2.0 (the "License");
  ~ you may not use this file except in compliance with the License.
  ~ You may obtain a copy of the License at
  ~
  ~     http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing, software
  ~ distributed under the License is distributed on an "AS IS" BASIS,
  ~ WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  ~ See the License for the specific language governing permissions and
  ~ limitations under the License.
-->

<script lang="ts">
  import type { ILink } from '$lib/interfaces/link';

  const icons: ILink[] = [
    {
      title: 'github',
      url: 'https://github.com/mrsimonemms/temporal-dsl',
    },
  ];

  const startYear = 2025;
  const currentYear = new Date().getFullYear();

  let copyrightYear = `${startYear}`;
  if (currentYear > startYear) {
    copyrightYear += ` - ${currentYear}`;
  }
</script>

<footer class="footer p-5 is-size-7">
  <div class="columns is-vcentered">
    <div class="column">
      <p>
        &copy; Simon Emms {copyrightYear} | Released under
        <a
          href="https://github.com/mrsimonemms/temporal-dsl/blob/main/LICENSE"
          target="_blank">Apache 2.0 licence</a
        >
      </p>
      <p>
        This is a community-maintained project and not associated with Temporal
        Technologies
      </p>
    </div>
    <div class="column has-text-right">
      {#each icons as item}
        <a
          href={item.url}
          style="color: currentColor"
          target="_blank"
          aria-label={item.title}
        >
          <span class="icon">
            <i class="mdi mdi-24px mdi-{item.title}"></i>
          </span>
        </a>
      {/each}
    </div>
  </div>
</footer>
