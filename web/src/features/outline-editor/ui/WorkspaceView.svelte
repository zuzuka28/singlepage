<script lang="ts">
  import BlockNode from './BlockNode.svelte';
  import type { OutlineDocument } from '../../../entities/outline';
  import type { DropPosition } from '../logic/editor';

  type SettingsModal = 'password' | 'rotate';

  export let document: OutlineDocument;
  export let query: string;
  export let searchOpen: boolean;
  export let suggestions: string[];
  export let saveState: 'saved' | 'saving' | 'error' | 'conflict' | 'revoked';
  export let theme: 'light' | 'dark';
  export let settingsOpen: boolean;
  export let linkCopied: boolean;
  export let linkCopyError: string;
  export let localSessionError: string;
  export let historyAvailable: boolean;
  export let autosaveStopped: boolean;
  export let focusedId: string | null;
  export let rootIds: string[];
  export let displayRootIds: string[];
  export let filteredIds: Set<string> | null;
  export let matchedIds: Set<string>;
  export let trail: string[];
  export let selectedIds: Set<string>;
  export let draggedId: string | null;
  export let dropTarget: { id: string; position: DropPosition } | null;
  export let importInput: HTMLInputElement;

  export let returnToStart: () => void | Promise<void>;
  export let updateQuery: (value: string) => void;
  export let handleSearchKeydown: (event: KeyboardEvent) => void;
  export let returnToEditor: () => void;
  export let chooseSuggestion: (suggestion: string) => void;
  export let toggleTheme: () => void;
  export let copyLink: () => void | Promise<void>;
  export let exportMarkdownFile: () => void;
  export let chooseMarkdownFile: () => void;
  export let openModal: (modal: SettingsModal) => void;
  export let importMarkdownFile: (event: Event) => void | Promise<void>;
  export let reload: () => void;
  export let focusBranch: (id: string | null) => void;
  export let addFirstChild: (id: string) => void;
  export let register: (id: string, node: HTMLTextAreaElement | null) => void;
  export let rememberEditor: (id: string) => void;
  export let updateText: (id: string, text: string) => void;
  export let handleBlockKeydown: (event: KeyboardEvent, id: string) => void;
  export let toggle: (id: string) => void;
  export let dragStart: (event: DragEvent, id: string) => void;
  export let dragOver: (event: DragEvent, id: string) => void;
  export let drop: (event: DragEvent, id: string) => void;
  export let dragEnd: () => void;
</script>

<main class="workspace">
  <header class="topbar">
    <button class="wordmark wordmark-button" type="button" aria-label="Back to start" title="Back to start" on:click={returnToStart}><span class="brand-mark">S</span><span>singlepage / outline</span></button>
    <div class="search-wrap">
      <span class="search-icon">⌕</span>
      <input
        class="search-input"
        aria-label="Search"
        placeholder="Search"
        value={query}
        on:input={(event) => updateQuery(event.currentTarget.value)}
        on:keydown={handleSearchKeydown}
        on:focus={() => searchOpen = true}
      />
      {#if query}
        <button class="search-clear" type="button" aria-label="Clear search" on:click={returnToEditor}>×</button>
      {:else}
        <span class="key-hint">⌘K</span>
      {/if}
      {#if searchOpen && suggestions.length}
        <div class="autocomplete-menu">
          <div class="suggestions">
            {#each suggestions as suggestion}<button type="button" on:click={() => chooseSuggestion(suggestion)}>{suggestion}</button>{/each}
          </div>
        </div>
      {/if}
    </div>
    <div class="status-actions">
      <span class="save-state">{saveState === 'saving' ? 'Saving…' : saveState === 'saved' ? 'Saved' : saveState === 'conflict' ? 'Changed elsewhere' : saveState === 'revoked' ? 'Link replaced' : 'Not saved'}</span>
      <button class="icon-button" type="button" aria-label={theme === 'light' ? 'Use dark theme' : 'Use light theme'} on:click={toggleTheme}>◐</button>
      <button class="icon-button" type="button" aria-label="Settings" on:click={() => settingsOpen = !settingsOpen}>•••</button>
    </div>
  </header>

  {#if settingsOpen}
    <div class="popover">
      <button type="button" on:click={returnToStart}>Back to start</button>
      <button type="button" on:click={copyLink}>{linkCopied ? 'Link copied' : 'Copy access link'}</button>
      {#if linkCopyError}<p class="popover-error" role="alert">{linkCopyError}</p>{/if}
      {#if historyAvailable && localSessionError}<p class="popover-error" role="alert">{localSessionError}</p>{/if}
      <hr />
      <button type="button" on:click={exportMarkdownFile}>Export Markdown</button>
      <button type="button" disabled={autosaveStopped} on:click={chooseMarkdownFile}>Import Markdown</button>
      <hr />
      <button type="button" on:click={() => openModal('password')}>Change password</button>
      <button type="button" on:click={() => openModal('rotate')}>Create new access link</button>
    </div>
  {/if}

  <input
    class="file-input"
    bind:this={importInput}
    type="file"
    accept=".md,text/markdown,text/plain"
    aria-label="Markdown file"
    on:change={importMarkdownFile}
  />

  {#if saveState === 'conflict'}
    <div class="conflict"><span>This page changed elsewhere.</span><button class="secondary" type="button" on:click={reload}>Reload page</button></div>
  {:else if saveState === 'revoked'}
    <div class="conflict"><span>This link has been replaced. Use the newest link to continue.</span></div>
  {/if}

  <section class="canvas">
    <nav class="breadcrumbs" aria-label="Focused branch">
      <button type="button" on:click={() => focusBranch(null)}>All notes</button>
      {#each trail as id}<span>›</span><button type="button" on:click={() => focusBranch(id)}>{document.blocks[id]?.text || 'Untitled'}</button>{/each}
    </nav>
    <div class="outline">
      {#each displayRootIds as id (id)}
        <BlockNode
          {document}
          blockId={id}
          visibleIds={filteredIds}
          {matchedIds}
          {selectedIds}
          {draggedId}
          {dropTarget}
          searching={searchOpen && Boolean(query.trim())}
          {register}
          focus={rememberEditor}
          {updateText}
          keydown={handleBlockKeydown}
          {toggle}
          {focusBranch}
          {dragStart}
          {dragOver}
          {drop}
          {dragEnd}
        />
      {/each}
      {#if focusedId && rootIds.length === 0}
        <button class="secondary" type="button" on:click={() => addFirstChild(focusedId!)}>Add the first child</button>
      {:else if query.trim() && displayRootIds.length === 0}
        <p class="empty-search">No matches</p>
      {/if}
    </div>
  </section>
</main>
