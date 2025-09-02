<script lang="ts">
  import {
    SvelteFlow,
    Background,
    MiniMap,
    Controls,
    Panel,
  } from '@xyflow/svelte';

  import '@xyflow/svelte/dist/style.css';

  let nodes = $state.raw([
    { id: '1', position: { x: 0, y: 0 }, data: { label: '1' } },
    { id: '2', position: { x: 0, y: 100 }, data: { label: '2' } },
  ]);

  let edges = $state.raw([{ id: 'e1-2', source: '1', target: '2' }]);
</script>

<SvelteFlow bind:nodes bind:edges fitView>
  <MiniMap />
  <Controls />
  <Background />
  <Panel position="top-left">
    <h1>hello</h1>
  </Panel>
</SvelteFlow>
