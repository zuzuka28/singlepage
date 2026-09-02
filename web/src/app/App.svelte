<script lang="ts">
  import type { ApplicationRuntime } from './logic/runtime';
  import { useApplicationActor } from './logic/app-actor.svelte';
  import AppShell from './AppShell.svelte';

  export let runtime: ApplicationRuntime;
  const application = useApplicationActor(runtime);
</script>

<svelte:head>
  <meta name="description" content="A private, encrypted infinite outline." />
</svelte:head>

<AppShell {runtime} {application} />
