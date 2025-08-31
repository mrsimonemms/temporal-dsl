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
  import iconLight from '$lib/assets/icon.png';
  import iconDark from '$lib/assets/icon-dark.png';

  let { isDark = false } = $props();

  const icon = isDark ? iconDark : iconLight;
</script>

<nav class="navbar" aria-label="main navigation dropdown">
  <div class="navbar-brand">
    <a class="navbar-item" href="/" aria-label="Home">
      <img src={icon} alt="Home" />
    </a>
  </div>

  <div class="navbar-menu">
    <div class="navbar-end">
      <a href="/" class="navbar-item" aria-label="Add">
        <span class="icon">
          <i class="mdi mdi-plus"></i>
        </span>
      </a>

      <div class="navbar-item has-dropdow s-hoverable">
        <a href="/" class="navbar-link">
          <figure class="image is-24x24">
            <img
              class="is-rounded"
              alt="avatar"
              src="https://bulma.io/assets/images/placeholders/24x24.png"
            />
          </figure>
        </a>
        <div class="navbar-dropdown">
          <a href="/" class="navbar-item">Components</a>
        </div>
      </div>
    </div>
  </div>
</nav>

<style lang="scss" scoped>
  nav {
    border: {
      bottom: {
        color: var(--bulma-grey-lighter);
        style: solid;
        width: 1px;
      }
    }
  }
</style>
