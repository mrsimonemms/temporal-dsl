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
  import favicon from '$lib/assets/icon.png';
  import Navbar from '$lib/components/navbar.svelte';
  import Footer from '$lib/components/footer.svelte';
  import Menu from '$lib/components/menu.svelte';

  import '../app.scss';

  let { children } = $props();
</script>

<svelte:head>
  <link rel="icon" href={favicon} />
</svelte:head>

<div class="columns is-flex-grow-1 is-gapless">
  <div class="sidebar column is-2 is-flex is-flex-direction-column">
    <div class="m-2 is-flex is-flex-direction-column is-flex-grow-1">
      <Menu />

      <div class="is-flex-grow-1"></div>

      <aside class="menu">
        <ul class="menu-list">
          <li><a href="/">Dashboard</a></li>
        </ul>
      </aside>
    </div>
  </div>
  <div class="column is-flex is-flex-direction-column">
    <Navbar />

    <div class="is-flex-grow-1 m-2">
      {@render children?.()}
    </div>

    <Footer />
  </div>
</div>

<style lang="scss" scoped>
  .columns.is-gapless:not(:last-child) {
    margin: {
      bottom: 0;
    }
  }

  .sidebar {
    border: {
      right: {
        color: var(--bulma-grey-lighter);
        style: solid;
        width: 1px;
      }
    }
  }
</style>
