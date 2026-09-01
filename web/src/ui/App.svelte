<script lang="ts">
  import { onMount, tick } from 'svelte';
  import BlockNode from './BlockNode.svelte';
  import {
    ancestors,
    appendSibling,
    buildAutocomplete,
    buildIndex,
    createDocument,
    descendants,
    indent,
    insertAfter,
    insertChild,
    insertRoot,
    outdent,
    parseMarkdown,
    removeBlock,
    reorder,
    search,
    serializeMarkdown,
    updateBlock,
    type OutlineDocument
  } from '../core';
  import { changePassword, decryptJson, encryptJson, generateSecret, generateWriteToken, randomBytes } from '../crypto';
  import { PageApi, RemoteApiError, RevisionConflictError, type RemotePage } from '../remote';

  type VaultDocument = { document: OutlineDocument; writeToken: string };
  type Screen = 'create' | 'loading' | 'unlock' | 'outline' | 'missing';
  type Modal = 'link' | 'password' | 'rotate' | null;

  const api = new PageApi();
  const minimumPasswordLength = 16;
  const editors = new Map<string, HTMLTextAreaElement>();
  let screen: Screen = 'create';
  let pageId = '';
  let urlSecret = '';
  let password = '';
  let passwordRepeat = '';
  let unlockPassword = '';
  let authError = '';
  let busy = false;
  let remotePage: RemotePage | null = null;
  let document: OutlineDocument = createDocument();
  let writeToken = '';
  let revision = 0;
  let focusedId: string | null = null;
  let lastEditorId: string | null = null;
  let query = '';
  let filterVisibleExtras = new Set<string>();
  let searchOpen = false;
  let settingsOpen = false;
  let theme: 'light' | 'dark' = 'light';
  let saveState: 'saved' | 'saving' | 'error' | 'conflict' | 'revoked' = 'saved';
  let modal: Modal = null;
  let newPassword = '';
  let newPasswordRepeat = '';
  let modalError = '';
  let saveTimer: ReturnType<typeof setTimeout> | undefined;
  let saveGeneration = 0;
  let savedGeneration = 0;
  let writing = false;
  let writeQueue: Promise<void> = Promise.resolve();
  let autosaveStopped = false;
  let importInput: HTMLInputElement;

  $: index = buildIndex(document);
  $: results = query.trim() ? search(index, query) : [];
  $: autocomplete = buildAutocomplete(index);
  $: suggestions = suggestionsFor(query);
  $: rootIds = focusedId ? (document.blocks[focusedId]?.children ?? []) : document.roots;
  $: matchedIds = new Set(results.map((result) => result.id));
  $: filteredIds = query.trim()
    ? new Set([
        ...results.flatMap((result) => [...result.ancestors, result.id, ...descendants(document, result.id)]),
        ...filterVisibleExtras
      ])
    : null;
  $: displayRootIds = rootIds.filter((id) => !filteredIds || filteredIds.has(id));
  $: trail = focusedId ? [...ancestors(document, focusedId), focusedId] : [];
  $: accessLink = pageId && urlSecret ? `${location.origin}/p/${pageId}#${urlSecret}` : '';

  onMount(() => {
    const savedTheme = localStorage.getItem('mindrop-theme');
    setTheme(savedTheme === 'light' || savedTheme === 'dark'
      ? savedTheme
      : matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light', false);
    void initialize();
    const shortcut = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && ['f', 'k'].includes(event.key.toLowerCase())) {
        event.preventDefault();
        searchOpen = true;
        tick().then(() => globalThis.document.querySelector<HTMLInputElement>('.search-input')?.focus());
      }
      if (event.ctrlKey && event.key === '[' && screen === 'outline' && focusedId) {
        event.preventDefault();
        zoomOut();
      }
      if (event.key === 'Escape') {
        const activeElement = globalThis.document.activeElement;
        if (activeElement instanceof HTMLElement && activeElement.closest('.search-wrap')) {
          event.preventDefault();
          returnToEditor();
        } else {
          if (query) query = '';
          searchOpen = false;
        }
        settingsOpen = false;
      }
    };
    const beforeUnload = (event: BeforeUnloadEvent) => {
      if (saveGeneration !== savedGeneration || writing) {
        event.preventDefault();
        event.returnValue = '';
      }
    };
    const hashChanged = () => void handleHashChange();
    window.addEventListener('keydown', shortcut);
    window.addEventListener('beforeunload', beforeUnload);
    window.addEventListener('hashchange', hashChanged);
    return () => {
      window.removeEventListener('keydown', shortcut);
      window.removeEventListener('beforeunload', beforeUnload);
      window.removeEventListener('hashchange', hashChanged);
    };
  });

  function setTheme(next: 'light' | 'dark', persist = true) {
    theme = next;
    globalThis.document.documentElement.dataset.theme = next;
    globalThis.document.querySelector<HTMLMetaElement>('meta[name="theme-color"]')?.setAttribute('content', next === 'dark' ? '#1d2027' : '#f7f8fb');
    if (persist) localStorage.setItem('mindrop-theme', next);
  }

  function toggleTheme() {
    setTheme(theme === 'light' ? 'dark' : 'light');
  }

  async function handleHashChange() {
    const nextSecret = location.hash.slice(1);
    if (!pageId || nextSecret === urlSecret) return;
    if (saveGeneration !== savedGeneration || writing) {
      history.replaceState({}, '', `/p/${pageId}#${urlSecret}`);
      return;
    }
    urlSecret = nextSecret;
    password = '';
    unlockPassword = '';
    writeToken = '';
    document = createDocument();
    focusedId = null;
    query = '';
    modal = null;
    settingsOpen = false;
    autosaveStopped = false;
    saveState = 'saved';
    remotePage = null;
    screen = 'loading';
    try {
      remotePage = await api.getPage(pageId);
      revision = remotePage.revision;
      screen = 'unlock';
    } catch {
      screen = 'missing';
    }
  }

  async function initialize() {
    const route = /^\/p\/([A-Za-z0-9_-]{16,128})\/?$/.exec(location.pathname);
    if (!route) {
      screen = 'create';
      return;
    }
    pageId = route[1];
    urlSecret = location.hash.slice(1);
    screen = 'loading';
    try {
      remotePage = await api.getPage(pageId);
      revision = remotePage.revision;
      screen = 'unlock';
    } catch {
      screen = 'missing';
    }
  }

  function register(id: string, node: HTMLTextAreaElement | null) {
    if (node) editors.set(id, node);
    else editors.delete(id);
  }

  async function focusEditor(id: string | undefined, caret: 'start' | 'end' | number = 'end') {
    if (!id) return;
    await tick();
    const editor = editors.get(id);
    editor?.focus();
    if (editor) {
      lastEditorId = id;
      const position = caret === 'start' ? 0 : caret === 'end' ? editor.value.length : Math.min(caret, editor.value.length);
      editor.setSelectionRange(position, position);
    }
  }

  function rememberEditor(id: string) {
    lastEditorId = id;
    searchOpen = false;
  }

  function returnToEditor() {
    query = '';
    filterVisibleExtras = new Set();
    searchOpen = false;
    const fallbackId = focusedId ? document.blocks[focusedId]?.children[0] : document.roots[0];
    void focusEditor(lastEditorId && document.blocks[lastEditorId] ? lastEditorId : fallbackId);
  }

  function updateQuery(value: string) {
    query = value;
    filterVisibleExtras = new Set();
    searchOpen = true;
  }

  async function enterFilteredTree() {
    const targetId = results[0]?.id;
    if (!targetId) return;
    let expanded = document;
    for (const blockId of [...ancestors(document, targetId), targetId]) {
      if (expanded.blocks[blockId]?.collapsed) expanded = updateBlock(expanded, blockId, { collapsed: false });
    }
    searchOpen = false;
    if (expanded !== document) setDocument(expanded);
    await focusEditor(targetId, 'end');
  }

  function handleSearchKeydown(event: KeyboardEvent) {
    if (event.key !== 'Enter' || event.isComposing) return;
    event.preventDefault();
    void enterFilteredTree();
  }

  function keepVisibleInFilter(id: string, source = document) {
    if (!query.trim()) return;
    filterVisibleExtras = new Set([...filterVisibleExtras, ...ancestors(source, id), id]);
  }

  async function createPage() {
    authError = '';
    if (password.length < minimumPasswordLength) { authError = `Use at least ${minimumPasswordLength} characters.`; return; }
    if (password !== passwordRepeat) { authError = 'Passwords do not match.'; return; }
    busy = true;
    try {
      pageId = generateSecret(16);
      urlSecret = generateSecret(32);
      writeToken = generateWriteToken(32);
      const first = insertRoot(createDocument());
      document = first.document;
      const encrypted = await encryptJson({ document, writeToken } satisfies VaultDocument, password, urlSecret);
      const created = await api.createPage({ id: pageId, salt: encrypted.salt, ciphertext: encrypted.ciphertext, writeToken });
      revision = created.revision;
      remotePage = { revision, salt: encrypted.salt, ciphertext: encrypted.ciphertext };
      history.replaceState({}, '', `/p/${pageId}#${urlSecret}`);
      screen = 'outline';
      modal = 'link';
      saveState = 'saved';
      await focusEditor(first.blockId);
    } catch (error) {
      console.error('Page creation failed', error);
      authError = 'Unable to create the page. Try again.';
    } finally {
      busy = false;
    }
  }

  async function unlockPage() {
    authError = '';
    if (!remotePage || !urlSecret) { authError = decryptMessage(); return; }
    busy = true;
    try {
      const vault = await decryptJson<VaultDocument>(remotePage, unlockPassword, urlSecret);
      if (!vault?.document || !vault.writeToken) throw new Error('Invalid document');
      document = vault.document;
      writeToken = vault.writeToken;
      password = unlockPassword;
      screen = 'outline';
      saveState = 'saved';
      if (!document.roots.length) {
        const created = insertRoot(document);
        document = created.document;
        markChanged();
        await focusEditor(created.blockId);
      } else {
        await focusEditor(document.roots[0]);
      }
    } catch {
      authError = decryptMessage();
    } finally {
      busy = false;
    }
  }

  function decryptMessage() {
    return 'Unable to open page. Check the password and link.';
  }

  function setDocument(next: OutlineDocument, focusId?: string) {
    document = next;
    markChanged();
    if (focusId) void focusEditor(focusId);
  }

  function updateText(id: string, text: string) {
    keepVisibleInFilter(id);
    setDocument(updateBlock(document, id, { text }));
  }

  function toggle(id: string) {
    searchOpen = false;
    const block = document.blocks[id];
    setDocument(updateBlock(document, id, { collapsed: !block.collapsed }));
  }

  function addFirstChild(parentId: string) {
    const added = insertChild(document, parentId);
    keepVisibleInFilter(added.blockId, added.document);
    setDocument(added.document, added.blockId);
  }

  function handleBlockKeydown(event: KeyboardEvent, id: string) {
    if (event.isComposing) return;
    const block = document.blocks[id];
    if (event.ctrlKey && event.key === ']') {
      event.preventDefault();
      focusBranch(id);
      return;
    }
    if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      addFirstChild(id);
      return;
    }
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      const isFilteredTopLevel = Boolean(query.trim()) && block.parentId === focusedId;
      const inserted = isFilteredTopLevel ? appendSibling(document, id) : insertAfter(document, id);
      keepVisibleInFilter(inserted.blockId, inserted.document);
      setDocument(inserted.document, inserted.blockId);
      return;
    }
    if (event.key === 'Tab') {
      event.preventDefault();
      setDocument(event.shiftKey ? outdent(document, id) : indent(document, id), id);
      return;
    }
    if (event.key === 'Backspace' && block.text === '' && block.children.length === 0) {
      const visible = displayedBlockIds();
      if (Object.keys(document.blocks).length === 1) return;
      event.preventDefault();
      const previous = visible[Math.max(0, visible.indexOf(id) - 1)];
      setDocument(removeBlock(document, id), previous);
      return;
    }
    if ((event.metaKey || event.ctrlKey) && (event.key === 'ArrowUp' || event.key === 'ArrowDown')) {
      event.preventDefault();
      const list = block.parentId ? document.blocks[block.parentId].children : document.roots;
      const current = list.indexOf(id);
      const target = event.key === 'ArrowUp' ? current - 1 : current + 1;
      setDocument(reorder(document, id, target), id);
      return;
    }
    if (!event.shiftKey && !event.altKey && !event.metaKey && !event.ctrlKey && (event.key === 'ArrowUp' || event.key === 'ArrowDown')) {
      const textarea = event.currentTarget as HTMLTextAreaElement;
      const collapsedSelection = textarea.selectionStart === textarea.selectionEnd;
      const atStart = textarea.selectionStart === 0;
      const atEnd = textarea.selectionStart === textarea.value.length;
      if (collapsedSelection && ((event.key === 'ArrowUp' && atStart) || (event.key === 'ArrowDown' && atEnd))) {
        const visible = displayedBlockIds();
        const current = visible.indexOf(id);
        const next = visible[current + (event.key === 'ArrowUp' ? -1 : 1)];
        if (next) {
          event.preventDefault();
          void focusEditor(next, event.key === 'ArrowUp' ? 'end' : 'start');
        }
      }
    }
  }

  function displayedBlockIds(): string[] {
    const ids: string[] = [];
    const visit = (id: string) => {
      if (filteredIds && !filteredIds.has(id)) return;
      ids.push(id);
      const block = document.blocks[id];
      if (query.trim() || !block.collapsed) block.children.forEach(visit);
    };
    rootIds.forEach(visit);
    return ids;
  }

  function markChanged() {
    if (autosaveStopped) return;
    saveGeneration += 1;
    saveState = 'saving';
    clearTimeout(saveTimer);
    saveTimer = setTimeout(() => void persist(), 650);
  }

  function serializeWrite<T>(operation: () => Promise<T>): Promise<T> {
    const result = writeQueue.then(operation, operation);
    writeQueue = result.then(() => undefined, () => undefined);
    return result;
  }

  async function persist() {
    clearTimeout(saveTimer);
    if (autosaveStopped || screen !== 'outline' || savedGeneration === saveGeneration) return;
    let retry = false;
    await serializeWrite(async () => {
      if (autosaveStopped || savedGeneration === saveGeneration) return;
      writing = true;
      const generation = saveGeneration;
      const snapshot = document;
      try {
        const encrypted = await encryptJson({ document: snapshot, writeToken } satisfies VaultDocument, password, urlSecret, remotePage?.salt ?? randomBytes(16));
        const updated = await api.updatePage(pageId, writeToken, {
          expectedRevision: revision,
          ciphertext: encrypted.ciphertext
        });
        revision = updated.revision;
        remotePage = { revision, salt: encrypted.salt, ciphertext: encrypted.ciphertext };
        savedGeneration = generation;
        saveState = generation === saveGeneration ? 'saved' : 'saving';
      } catch (error) {
        if (error instanceof RevisionConflictError) {
          autosaveStopped = true;
          saveState = 'conflict';
        } else if (error instanceof RemoteApiError && (error.status === 401 || error.status === 403 || error.status === 404)) {
          autosaveStopped = true;
          saveState = 'revoked';
        } else {
          retry = true;
          saveState = 'error';
        }
      } finally {
        writing = false;
      }
    });
    if (!autosaveStopped && savedGeneration !== saveGeneration) {
      saveTimer = setTimeout(() => void persist(), retry ? 2500 : 100);
    }
  }

  function focusBranch(id: string | null) {
    focusedId = id;
    searchOpen = false;
    tick().then(() => focusEditor(rootIds[0]));
  }

  function zoomOut() {
    if (!focusedId) return;
    const previousFocus = focusedId;
    focusedId = document.blocks[previousFocus]?.parentId ?? null;
    searchOpen = false;
    tick().then(() => focusEditor(previousFocus));
  }

  function suggestionsFor(value: string): string[] {
    if (!value) return [];
    const token = value.split(/\s+/).at(-1) ?? '';
    const normalized = token.toLocaleLowerCase();
    if (normalized.startsWith('#')) return autocomplete.tags.filter((tag) => tag.startsWith(normalized.slice(1))).slice(0, 6).map((tag) => `#${tag}`).filter((item) => item !== normalized);
    if (normalized.startsWith('@')) return ['@2026-09-01', '@>=2026-09-01', '@2026-09-01..2026-09-30'].filter((item) => item !== normalized);
    const colon = normalized.indexOf(':');
    if (colon >= 0) {
      const key = normalized.slice(0, colon);
      const prefix = normalized.slice(colon + 1);
      return (autocomplete.properties[key] ?? []).filter((item) => item.toLocaleLowerCase().startsWith(prefix)).slice(0, 6).map((item) => `${key}:${item}`).filter((item) => item.toLocaleLowerCase() !== normalized);
    }
    return Object.keys(autocomplete.properties).filter((key) => key.startsWith(normalized)).slice(0, 6).map((key) => `${key}:`);
  }

  function chooseSuggestion(suggestion: string) {
    const parts = query.split(/\s+/);
    parts[parts.length - 1] = suggestion;
    query = `${parts.join(' ')} `;
    filterVisibleExtras = new Set();
    searchOpen = true;
  }

  async function copyLink() {
    await navigator.clipboard.writeText(accessLink);
  }

  function exportMarkdownFile() {
    settingsOpen = false;
    const blob = new Blob([serializeMarkdown(document)], { type: 'text/markdown;charset=utf-8' });
    const href = URL.createObjectURL(blob);
    const link = globalThis.document.createElement('a');
    link.href = href;
    link.download = `mindrop-outline-${new Date().toISOString().slice(0, 10)}.md`;
    link.click();
    setTimeout(() => URL.revokeObjectURL(href), 0);
  }

  function chooseMarkdownFile() {
    settingsOpen = false;
    if (autosaveStopped) {
      alert('Resolve the saving issue before importing a file.');
      return;
    }
    importInput.value = '';
    importInput.click();
  }

  async function importMarkdownFile(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file) return;

    const hasContent = Object.values(document.blocks).some((block) => block.text || block.children.length) || document.roots.length > 1;
    if (hasContent && !confirm('Importing this file will replace all current notes. Continue?')) return;

    try {
      let imported = parseMarkdown(await file.text());
      if (!imported.roots.length) imported = insertRoot(imported).document;
      focusedId = null;
      lastEditorId = null;
      query = '';
      filterVisibleExtras = new Set();
      searchOpen = false;
      setDocument(imported, imported.roots[0]);
    } catch (error) {
      console.error('Markdown import failed', error);
      alert('Unable to import this Markdown file.');
    }
  }

  async function submitPasswordChange() {
    modalError = '';
    if (newPassword.length < minimumPasswordLength) { modalError = `Use at least ${minimumPasswordLength} characters.`; return; }
    if (newPassword !== newPasswordRepeat) { modalError = 'Passwords do not match.'; return; }
    busy = true;
    saveState = 'saving';
    clearTimeout(saveTimer);
    try {
      await serializeWrite(async () => {
        writing = true;
        try {
          const generation = saveGeneration;
          const nextWriteToken = generateWriteToken(32);
          const encrypted = await changePassword({ document, writeToken: nextWriteToken }, newPassword, urlSecret);
          const updated = await api.updatePage(pageId, writeToken, {
            expectedRevision: revision,
            ciphertext: encrypted.ciphertext,
            salt: encrypted.salt,
            newWriteToken: nextWriteToken
          });
          password = newPassword;
          writeToken = nextWriteToken;
          revision = updated.revision;
          remotePage = { revision, salt: encrypted.salt, ciphertext: encrypted.ciphertext };
          savedGeneration = generation;
          modal = null;
          newPassword = newPasswordRepeat = '';
          saveState = generation === saveGeneration ? 'saved' : 'saving';
        } finally { writing = false; }
      });
    } catch (error) {
      handleSettingsError(error);
    } finally { busy = false; }
    if (!autosaveStopped && savedGeneration !== saveGeneration) saveTimer = setTimeout(() => void persist(), 100);
  }

  async function rotateLink() {
    modalError = '';
    busy = true;
    saveState = 'saving';
    clearTimeout(saveTimer);
    const nextPageId = generateSecret(16);
    const nextSecret = generateSecret(32);
    const nextToken = generateWriteToken(32);
    try {
      await serializeWrite(async () => {
        writing = true;
        try {
          const generation = saveGeneration;
          const encrypted = await encryptJson({ document, writeToken: nextToken }, password, nextSecret);
          const updated = await api.rotatePage(pageId, writeToken, {
            newId: nextPageId,
            ciphertext: encrypted.ciphertext,
            salt: encrypted.salt,
            newWriteToken: nextToken
          });
          pageId = nextPageId;
          urlSecret = nextSecret;
          writeToken = nextToken;
          revision = updated.revision;
          remotePage = { revision, salt: encrypted.salt, ciphertext: encrypted.ciphertext };
          savedGeneration = generation;
          history.replaceState({}, '', `/p/${pageId}#${urlSecret}`);
          modal = 'link';
          saveState = generation === saveGeneration ? 'saved' : 'saving';
        } finally { writing = false; }
      });
    } catch (error) {
      handleSettingsError(error);
    } finally { busy = false; }
    if (!autosaveStopped && savedGeneration !== saveGeneration) saveTimer = setTimeout(() => void persist(), 100);
  }

  function handleSettingsError(error: unknown) {
    if (error instanceof RevisionConflictError) {
      autosaveStopped = true;
      saveState = 'conflict';
      modal = null;
    } else if (error instanceof RemoteApiError && (error.status === 401 || error.status === 403 || error.status === 404)) {
      autosaveStopped = true;
      saveState = 'revoked';
      modal = null;
    } else modalError = 'Unable to update this page. Try again.';
  }

  function openModal(value: Modal) {
    settingsOpen = false;
    modalError = '';
    modal = value;
  }
</script>

<svelte:head>
  <meta name="description" content="A private, encrypted infinite outline." />
</svelte:head>

<div class="shell">
  {#if screen === 'create' || screen === 'unlock' || screen === 'loading' || screen === 'missing'}
    <header class="topbar auth-topbar">
      <div class="wordmark"><span class="brand-mark">M</span><span>mindrop / outline</span></div>
      <div class="auth-actions">
        <button class="icon-button" type="button" aria-label={theme === 'light' ? 'Use dark theme' : 'Use light theme'} on:click={toggleTheme}>◐</button>
      </div>
    </header>
    <main class="landing">
      <section class="brand-panel">
        <h1>Everything<br /><em>in one place.</em></h1>
        <p class="brand-note">Write first. Organize when it helps.</p>
      </section>
      <section class="form-panel">
        <form class="auth-card" on:submit|preventDefault={screen === 'create' ? createPage : unlockPage}>
          <div class="eyebrow">{screen === 'create' ? 'Your page' : 'Welcome back'}</div>
          {#if screen === 'loading'}
            <h2>Opening page…</h2>
          {:else if screen === 'missing'}
            <h2>Page not found</h2><p>Check the link and try again.</p>
          {:else if screen === 'create'}
            <h2>Create your page</h2>
            <p>Choose a password and keep the link you receive.</p>
            <label class="field"><span>Password</span><input type="password" autocomplete="new-password" minlength={minimumPasswordLength} bind:value={password} required /></label>
            <label class="field"><span>Repeat password</span><input type="password" autocomplete="new-password" minlength={minimumPasswordLength} bind:value={passwordRepeat} required /></label>
            <button class="primary" disabled={busy}>{busy ? 'Creating…' : 'Create page'}</button>
          {:else}
            <h2>Unlock page</h2>
            <p>Enter the password for this page.</p>
            <label class="field"><span>Password</span><input type="password" autocomplete="current-password" bind:value={unlockPassword} required /></label>
            <button class="primary" disabled={busy}>{busy ? 'Opening…' : 'Open'}</button>
          {/if}
          {#if authError}<p class="error" role="alert">{authError}</p>{/if}
        </form>
      </section>
    </main>
  {:else}
    <main class="workspace">
      <header class="topbar">
        <div class="wordmark"><span class="brand-mark">M</span><span>mindrop / outline</span></div>
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
          <button type="button" on:click={copyLink}>Copy access link</button>
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
        <div class="conflict"><span>This page changed elsewhere.</span><button class="secondary" type="button" on:click={() => location.reload()}>Reload page</button></div>
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
              searching={searchOpen && Boolean(query.trim())}
              {register}
              focus={rememberEditor}
              {updateText}
              keydown={handleBlockKeydown}
              {toggle}
              {focusBranch}
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
  {/if}

  {#if modal === 'link'}
    <div class="modal-backdrop" role="presentation">
      <div class="modal" role="dialog" aria-modal="true" aria-labelledby="link-title">
        <div class="eyebrow">Access link</div>
        <h2 id="link-title">Save this link.</h2>
        <p class="secret-link">{accessLink}</p>
        <p>You will also need your password.</p>
        <div class="button-row"><button class="primary" type="button" on:click={copyLink}>Copy link</button><button class="secondary" type="button" on:click={() => modal = null}>Done</button></div>
      </div>
    </div>
  {:else if modal === 'password'}
    <div class="modal-backdrop" role="presentation">
      <div class="modal" role="dialog" aria-modal="true" aria-labelledby="password-title">
        <form on:submit|preventDefault={submitPasswordChange}>
        <div class="eyebrow">Password</div><h2 id="password-title">Change password</h2><p>Use the new password the next time you open this page.</p>
        <label class="field"><span>New password</span><input type="password" autocomplete="new-password" minlength={minimumPasswordLength} bind:value={newPassword} required /></label>
        <label class="field"><span>Repeat new password</span><input type="password" autocomplete="new-password" minlength={minimumPasswordLength} bind:value={newPasswordRepeat} required /></label>
        {#if modalError}<p class="error">{modalError}</p>{/if}
        <div class="button-row"><button class="primary" disabled={busy}>{busy ? 'Changing…' : 'Change password'}</button><button class="secondary" type="button" on:click={() => modal = null}>Cancel</button></div>
        </form>
      </div>
    </div>
  {:else if modal === 'rotate'}
    <div class="modal-backdrop" role="presentation">
      <div class="modal" role="dialog" aria-modal="true" aria-labelledby="rotate-title">
        <div class="eyebrow">Access link</div><h2 id="rotate-title">Create a new link?</h2>
        <p>The old link will stop opening this page. Pages already open in another tab stay visible there until refreshed, but they can no longer save changes.</p>
        {#if modalError}<p class="error">{modalError}</p>{/if}
        <div class="button-row"><button class="primary" type="button" disabled={busy} on:click={rotateLink}>{busy ? 'Creating…' : 'Create new link'}</button><button class="secondary" type="button" on:click={() => modal = null}>Cancel</button></div>
      </div>
    </div>
  {/if}
</div>
