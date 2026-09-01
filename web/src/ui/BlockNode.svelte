<script lang="ts">
  import { afterUpdate, onMount } from 'svelte';
  import type { OutlineDocument } from '../core';

  export let document: OutlineDocument;
  export let blockId: string;
  export let highlightedId: string | null = null;
  export let visibleIds: Set<string> | null = null;
  export let matchedIds: Set<string> | null = null;
  export let searching = false;
  export let register: (id: string, node: HTMLTextAreaElement | null) => void;
  export let focus: (id: string) => void;
  export let updateText: (id: string, text: string) => void;
  export let keydown: (event: KeyboardEvent, id: string) => void;
  export let toggle: (id: string) => void;
  export let focusBranch: (id: string) => void;

  let textarea: HTMLTextAreaElement;
  $: block = document.blocks[blockId];
  $: childIds = block ? block.children.filter((id) => !visibleIds || visibleIds.has(id)) : [];

  function resize() {
    if (!textarea) return;
    textarea.style.height = '0px';
    textarea.style.height = `${Math.max(36, textarea.scrollHeight)}px`;
  }

  onMount(() => {
    register(blockId, textarea);
    resize();
    return () => register(blockId, null);
  });
  afterUpdate(resize);
</script>

{#if block && (!visibleIds || visibleIds.has(blockId))}
  <div class:highlighted={highlightedId === blockId} data-block-id={blockId}>
    <div class="block-row" class:search-match={matchedIds?.has(blockId)}>
      <div class="block-gutter">
        {#if block.children.length}
          <button
            type="button"
            class="disclosure"
            aria-label={block.collapsed ? 'Expand branch' : 'Collapse branch'}
            on:click={() => toggle(blockId)}
          >{block.collapsed ? '›' : '⌄'}</button>
        {:else}
          <span class="disclosure-spacer" aria-hidden="true"></span>
        {/if}
        <button
          type="button"
          class="block-bullet"
          class:has-children={block.children.length > 0}
          aria-label="Open branch"
          on:click={() => focusBranch(blockId)}
        ><span></span></button>
      </div>
      <textarea
        bind:this={textarea}
        class="block-text"
        rows="1"
        value={block.text}
        aria-label="Outline block"
        placeholder={block.parentId === null && document.roots.length === 1 ? 'Start writing…' : ''}
        on:focus={() => focus(blockId)}
        on:input={(event) => updateText(blockId, event.currentTarget.value)}
        on:keydown={(event) => keydown(event, blockId)}
      ></textarea>
    </div>
    {#if (searching || !block.collapsed) && childIds.length}
      <div class="children">
        {#each childIds as childId (childId)}
          <svelte:self
            {document}
            blockId={childId}
            {highlightedId}
            {visibleIds}
            {matchedIds}
            {searching}
            {register}
            {focus}
            {updateText}
            {keydown}
            {toggle}
            {focusBranch}
          />
        {/each}
      </div>
    {/if}
  </div>
{/if}
