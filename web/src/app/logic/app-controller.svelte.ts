import { onMount, tick } from 'svelte';
import {
  createDocument,
  insertChild,
  insertRoot,
  parseMarkdown,
  serializeMarkdown,
  updateBlock,
  type OutlineDocument
} from '../../entities/outline';
import {
  expandPath,
  expandSelection,
  indentSelected,
  interpretEditorKey,
  keepVisible,
  moveSelectedRelative,
  outdentSelected,
  removeSelectedBlocks,
  selectOutlineView,
  type DropPosition,
} from '../../features/outline-editor/logic/editor';
import {
  changeSessionPassword,
  createSession,
  identityFor,
  rotateSessionLink,
  unlockSession,
} from '../../features/page-session/logic/use-cases';
import { ApplicationError } from '../../features/page-session/logic/errors';
import type { AppViewState, OpenSession } from '../../features/page-session/logic/model';
import { parsePageLink } from '../../features/page-session/logic/page-link';
import { passwordValidationError } from '../../features/page-session/logic/password';
import type { Modal, Screen, StartMode } from '../../features/page-session/logic/view-model';
import type { ApplicationActorBinding } from './app-actor.svelte';
import type { ApplicationRuntime } from './runtime';

function screenFor(view: AppViewState): Screen {
  switch (view.kind) {
    case 'start': return 'create';
    case 'loading': return 'loading';
    case 'locked': return 'unlock';
    case 'missing': return 'missing';
    case 'open': return 'outline';
  }
}

interface HistoryEntry {
  document: OutlineDocument;
  focusId: string | null;
}

const HISTORY_LIMIT = 50;

export function createAppController(runtime: ApplicationRuntime, application: ApplicationActorBinding) {
  const sessionServices = {
    repository: runtime.repository,
    crypto: runtime.crypto,
    secrets: runtime.secrets,
    history: runtime.history,
  };
  const editors = new Map<string, HTMLTextAreaElement>();
  const emptyDocument = createDocument();
  let appView = $state.raw<AppViewState>(application.getView());
  let screen = $derived<Screen>(screenFor(appView));
  let session = $derived(appView.kind === 'open' ? appView.session : null);
  let identity = $derived(
    appView.kind === 'open'
      ? appView.session
      : appView.kind === 'loading' || appView.kind === 'locked' || appView.kind === 'missing'
        ? appView.identity
        : null,
  );
  let pageId = $derived(identity?.pageId ?? '');
  let urlSecret = $derived(identity?.urlSecret ?? '');
  let password = $state('');
  let passwordRepeat = $state('');
  let unlockPassword = $state('');
  let authError = $state('');
  let busy = $state(false);
  let remotePage = $derived(appView.kind === 'locked' ? appView.remotePage : session?.remotePage ?? null);
  let document = $derived<OutlineDocument>(session?.document ?? emptyDocument);
  let focusedId = $state<string | null>(null);
  let selectedIds = $state.raw(new Set<string>());
  let draggedId = $state<string | null>(null);
  let dropTarget = $state.raw<{ id: string; position: DropPosition } | null>(null);
  let lastEditorId: string | null = null;
  let undoStack: HistoryEntry[] = [];
  let redoStack: HistoryEntry[] = [];
  let activeHistoryGroup: string | null = null;
  let query = $state('');
  let filterVisibleExtras = $state.raw(new Set<string>());
  let searchOpen = $state(false);
  let settingsOpen = $state(false);
  let theme = $state<'light' | 'dark'>('light');
  let saveState = $derived(appView.kind === 'open' ? appView.saveState : 'saved');
  let modal = $state<Modal>(null);
  let newPassword = $state('');
  let newPasswordRepeat = $state('');
  let modalError = $state('');
  let writing = false;
  let autosaveStopped = $derived(saveState === 'conflict' || saveState === 'revoked');
  let importInput: HTMLInputElement;
  const historyAvailable = runtime.history.available;
  let localSessionError = $state(runtime.sessionError);
  let pageLinkInput = $state('');
  let pageLinkError = $state('');
  let nativeLocators = $state.raw<string[]>([]);
  let linkCopied = $state(false);
  let linkCopyError = $state('');
  let startMode = $state<StartMode>('create');

  let outlineView = $derived(selectOutlineView(document, focusedId, query, filterVisibleExtras));
  let results = $derived(outlineView.results);
  let suggestions = $derived(outlineView.suggestions);
  let rootIds = $derived(outlineView.rootIds);
  let matchedIds = $derived(outlineView.matchedIds);
  let filteredIds = $derived(outlineView.filteredIds);
  let displayRootIds = $derived(outlineView.displayRootIds);
  let trail = $derived(outlineView.trail);
  let pageLocator = $derived(pageId && urlSecret ? `/p/${pageId}#${urlSecret}` : '');
  let accessLink = $derived(pageLocator ? runtime.access.present(pageLocator) : '');

  onMount(() => {
    setTheme(runtime.theme.read() ?? 'light', false);
    const unsubscribe = application.view.subscribe((view) => {
      appView = view;
      syncLifecycleView(view);
    });
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
        if (selectedIds.size) {
          event.preventDefault();
          clearSelection();
          return;
        }
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
      if (application.isDirty() || writing) {
        event.preventDefault();
        event.returnValue = '';
      }
    };
    const hashChanged = () => void handleHashChange();
    const clearSelectionOnClick = () => {
      activeHistoryGroup = null;
      if (selectedIds.size) clearSelection();
    };
    window.addEventListener('keydown', shortcut);
    globalThis.document.addEventListener('click', clearSelectionOnClick);
    window.addEventListener('beforeunload', beforeUnload);
    window.addEventListener('hashchange', hashChanged);
    return () => {
      window.removeEventListener('keydown', shortcut);
      globalThis.document.removeEventListener('click', clearSelectionOnClick);
      window.removeEventListener('beforeunload', beforeUnload);
      window.removeEventListener('hashchange', hashChanged);
      unsubscribe();
    };
  });

  async function rememberLocalLocator() {
    if (!historyAvailable || !pageId || !urlSecret) return false;
    const remembered = await runtime.rememberLocator(`/p/${pageId}#${urlSecret}`);
    localSessionError = runtime.sessionError;
    return remembered;
  }

  async function refreshNativeLocators() {
    if (!historyAvailable) return;
    nativeLocators = await runtime.listLocators();
    localSessionError = runtime.sessionError;
    if (screen === 'create' && nativeLocators.length && !password && !pageLinkInput) startMode = 'open';
  }

  function setStartMode(mode: StartMode) {
    startMode = mode;
    authError = '';
    pageLinkError = '';
  }

  function resetPageState() {
    runtime.navigation.replace('/');
    application.send({ type: 'RESET' });
    password = '';
    passwordRepeat = '';
    unlockPassword = '';
    authError = '';
    pageLinkInput = '';
    pageLinkError = '';
    linkCopyError = '';
    startMode = historyAvailable && nativeLocators.length ? 'open' : 'create';
    focusedId = null;
    selectedIds = new Set();
    clearDragState();
    clearHistory();
    query = '';
    modal = null;
    settingsOpen = false;
  }

  async function returnToStart() {
    settingsOpen = false;
    if (screen === 'outline' && (application.isDirty() || writing)) {
      await persist();
      if (application.isDirty() || writing) {
        alert('Wait until this page is saved before returning to the start screen.');
        return;
      }
    }

    if (historyAvailable) {
      const remembered = await runtime.rememberLocator('/');
      localSessionError = runtime.sessionError;
      if (!remembered) {
        settingsOpen = screen === 'outline';
        return;
      }
    }
    resetPageState();
    await refreshNativeLocators();
  }

  function setTheme(next: 'light' | 'dark', persist = true) {
    theme = next;
    runtime.theme.apply(next, persist);
  }

  function toggleTheme() {
    setTheme(theme === 'light' ? 'dark' : 'light');
  }

  async function handleHashChange() {
    const nextSecret = location.hash.slice(1);
    if (!pageId || nextSecret === urlSecret) return;
    if (application.isDirty() || writing) {
      runtime.navigation.replace(`/p/${pageId}#${urlSecret}`);
      return;
    }
    prepareNavigation();
    application.send({ type: 'NAVIGATE', locator: `/p/${pageId}#${nextSecret}` });
  }

  async function openLocator(locator: string) {
    if (busy) return;
    const parsed = parsePageLink(locator, location.href);
    if (!parsed) {
      pageLinkError = 'Paste a valid Singlepage link.';
      return;
    }

    busy = true;
    pageLinkError = '';
    runtime.navigation.replace(parsed.locator);
    prepareNavigation();
    application.send({ type: 'NAVIGATE', locator: parsed.locator });
  }

  async function openPageLink() {
    if (busy) return;
    const parsed = parsePageLink(pageLinkInput, location.href);
    if (!parsed) {
      pageLinkError = 'Paste a valid Singlepage link.';
      return;
    }
    if (!historyAvailable && parsed.origin !== location.origin) {
      runtime.navigation.assign(parsed.href);
      return;
    }

    await openLocator(parsed.locator);
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
    if (lastEditorId !== id) activeHistoryGroup = null;
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
    const expanded = expandPath(document, targetId);
    searchOpen = false;
    if (expanded.document !== document) setDocument(expanded.document);
    await focusEditor(expanded.focusIntent?.id, expanded.focusIntent?.caret);
  }

  function handleSearchKeydown(event: KeyboardEvent) {
    if (event.key !== 'Enter' || event.isComposing) return;
    event.preventDefault();
    void enterFilteredTree();
  }

  function keepVisibleInFilter(id: string, source = document) {
    if (!query.trim()) return;
    filterVisibleExtras = keepVisible(source, id, filterVisibleExtras);
  }

  async function createPage() {
    authError = '';
    const validationError = passwordValidationError(password);
    if (validationError) { authError = validationError; return; }
    if (password !== passwordRepeat) { authError = 'Passwords do not match.'; return; }
    busy = true;
    try {
      const createdSession = await createSession(sessionServices, { password, epoch: application.nextEpoch() });
      application.send({ type: 'CREATED', session: createdSession });
      password = passwordRepeat = '';
      runtime.navigation.replace(createdSession.locator);
      modal = 'link';
      await focusEditor(createdSession.document.roots[0]);
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
      const session = await unlockSession(
        sessionServices,
        identityFor(pageId, urlSecret),
        remotePage,
        unlockPassword,
        application.nextEpoch(),
      );
      application.send({ type: 'OPENED', session });
      unlockPassword = '';
      if (!session.document.roots.length) {
        const created = insertRoot(session.document);
        setDocument(created.document);
        await focusEditor(created.blockId);
      } else {
        await focusEditor(session.document.roots[0]);
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

  function setDocument(next: OutlineDocument, focusId?: string, historyGroup: string | null = null, recordHistory = true) {
    if (next === document) return;
    if (recordHistory) {
      if (!historyGroup || activeHistoryGroup !== historyGroup) {
        undoStack = [...undoStack.slice(-(HISTORY_LIMIT - 1)), { document, focusId: lastEditorId }];
      }
      redoStack = [];
    }
    activeHistoryGroup = historyGroup;
    selectedIds = new Set([...selectedIds].filter((id) => Boolean(next.blocks[id])));
    application.send({ type: 'EDIT', document: next });
    if (focusId) void focusEditor(focusId);
  }

  function undo() {
    const entry = undoStack.at(-1);
    if (!entry) return;
    undoStack = undoStack.slice(0, -1);
    redoStack = [...redoStack.slice(-(HISTORY_LIMIT - 1)), { document, focusId: lastEditorId }];
    applyHistory(entry);
  }

  function redo() {
    const entry = redoStack.at(-1);
    if (!entry) return;
    redoStack = redoStack.slice(0, -1);
    undoStack = [...undoStack.slice(-(HISTORY_LIMIT - 1)), { document, focusId: lastEditorId }];
    applyHistory(entry);
  }

  function applyHistory(entry: HistoryEntry) {
    activeHistoryGroup = null;
    selectedIds = new Set();
    clearDragState();
    if (focusedId && !entry.document.blocks[focusedId]) focusedId = null;
    application.send({ type: 'EDIT', document: entry.document });
    const focusId = entry.focusId && entry.document.blocks[entry.focusId]
      ? entry.focusId
      : entry.document.roots[0];
    void focusEditor(focusId);
  }

  function clearHistory() {
    undoStack = [];
    redoStack = [];
    activeHistoryGroup = null;
  }

  function clearSelection() {
    selectedIds = new Set();
  }

  function deleteSelected() {
    if (!selectedIds.size) return;
    const visibleBefore = outlineView.visibleIds;
    const firstSelectedIndex = visibleBefore.findIndex((id) => selectedIds.has(id));
    let next = removeSelectedBlocks(document, selectedIds);
    selectedIds = new Set();

    if (!Object.keys(next.blocks).length) {
      const created = insertRoot(next);
      focusedId = null;
      next = created.document;
      setDocument(next, created.blockId);
      return;
    }

    if (focusedId && !next.blocks[focusedId]) focusedId = null;
    const remainingVisible = visibleBefore.filter((id) => Boolean(next.blocks[id]));
    const fallbackIndex = Math.max(0, Math.min(firstSelectedIndex - 1, remainingVisible.length - 1));
    setDocument(next, remainingVisible[fallbackIndex]);
  }

  function dragStart(event: DragEvent, id: string) {
    if (query.trim()) {
      event.preventDefault();
      return;
    }
    draggedId = id;
    dropTarget = null;
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'move';
      event.dataTransfer.setData('text/plain', id);
    }
  }

  function dragOver(event: DragEvent, id: string) {
    if (!draggedId || query.trim()) return;
    const row = event.currentTarget as HTMLElement;
    const rect = row.getBoundingClientRect();
    const ratio = rect.height ? (event.clientY - rect.top) / rect.height : 0.5;
    const position: DropPosition = ratio < 0.28 ? 'before' : ratio > 0.72 ? 'after' : 'inside';
    const movingIds = selectedIds.has(draggedId) ? selectedIds : new Set([draggedId]);
    if (!moveSelectedRelative(document, movingIds, id, position)) return;
    event.preventDefault();
    event.stopPropagation();
    if (event.dataTransfer) event.dataTransfer.dropEffect = 'move';
    dropTarget = { id, position };
  }

  function drop(event: DragEvent, id: string) {
    event.preventDefault();
    event.stopPropagation();
    const sourceId = draggedId ?? event.dataTransfer?.getData('text/plain');
    const position = dropTarget?.id === id ? dropTarget.position : null;
    if (!sourceId || !position) {
      clearDragState();
      return;
    }
    const movingIds = selectedIds.has(sourceId) ? selectedIds : new Set([sourceId]);
    const moved = moveSelectedRelative(document, movingIds, id, position);
    clearDragState();
    if (moved && moved !== document) setDocument(moved, sourceId);
  }

  function clearDragState() {
    draggedId = null;
    dropTarget = null;
  }

  function updateText(id: string, text: string) {
    keepVisibleInFilter(id);
    setDocument(updateBlock(document, id, { text }), undefined, `text:${id}`);
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
    if ((event.metaKey || event.ctrlKey) && (event.key.toLowerCase() === 'z' || event.code === 'KeyZ')) {
      event.preventDefault();
      event.shiftKey ? redo() : undo();
      return;
    }
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'a') {
      event.preventDefault();
      selectedIds = expandSelection(document, id, selectedIds);
      return;
    }
    if (selectedIds.has(id) && (event.key === 'Delete' || event.key === 'Backspace')) {
      event.preventDefault();
      deleteSelected();
      return;
    }
    if (event.key === 'Tab' && selectedIds.has(id)) {
      event.preventDefault();
      const shifted = event.shiftKey ? outdentSelected(document, selectedIds) : indentSelected(document, selectedIds);
      if (shifted !== document) setDocument(shifted, id);
      return;
    }
    if (event.ctrlKey && event.key === ']') {
      event.preventDefault();
      focusBranch(id);
      return;
    }
    const textarea = event.currentTarget as HTMLTextAreaElement;
    const result = interpretEditorKey(
      { document, id, focusedId, query, visibleIds: outlineView.visibleIds },
      {
        key: event.key,
        ctrl: event.ctrlKey,
        meta: event.metaKey,
        shift: event.shiftKey,
        alt: event.altKey,
        composing: event.isComposing,
        selectionStart: textarea.selectionStart,
        selectionEnd: textarea.selectionEnd,
      },
    );
    if (!result) return;
    event.preventDefault();
    if (result.document !== document) {
      if (result.focusIntent) keepVisibleInFilter(result.focusIntent.id, result.document);
      setDocument(result.document);
    }
    if (result.focusIntent) void focusEditor(result.focusIntent.id, result.focusIntent.caret);
  }

  async function persist() {
    if (autosaveStopped || screen !== 'outline') return;
    await application.flush();
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

  function chooseSuggestion(suggestion: string) {
    const parts = query.split(/\s+/);
    parts[parts.length - 1] = suggestion;
    query = `${parts.join(' ')} `;
    filterVisibleExtras = new Set();
    searchOpen = true;
  }

  async function copyLink() {
    linkCopyError = '';
    try {
      await runtime.clipboard.writeText(accessLink);
    } catch {
      if (!copyWithSelection(accessLink)) {
        linkCopyError = 'Unable to copy the link. Select it and copy it manually.';
        return;
      }
    }
    linkCopied = true;
    setTimeout(() => linkCopied = false, 1600);
  }

  function copyWithSelection(value: string): boolean {
    const input = globalThis.document.createElement('textarea');
    try {
      input.value = value;
      input.setAttribute('readonly', '');
      input.style.position = 'fixed';
      input.style.opacity = '0';
      globalThis.document.body.append(input);
      input.select();

      return globalThis.document.execCommand('copy');
    } catch {
      return false;
    } finally {
      input.remove();
    }
  }

  function exportMarkdownFile() {
    settingsOpen = false;
    runtime.files.exportMarkdown(
      serializeMarkdown(document),
      `singlepage-outline-${new Date().toISOString().slice(0, 10)}.md`,
    );
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
    const validationError = passwordValidationError(newPassword);
    if (validationError) { modalError = validationError; return; }
    if (newPassword !== newPasswordRepeat) { modalError = 'Passwords do not match.'; return; }
    busy = true;
    try {
      await persist();
      if (application.isDirty()) {
        modalError = 'Wait until this page is saved before changing the password.';
        return;
      }
      writing = true;
      const updated = await changeSessionPassword(sessionServices, currentSession(), document, newPassword);
      application.send({ type: 'CREATED', session: updated });
      modal = null;
      newPassword = newPasswordRepeat = '';
    } catch (error) {
      handleSettingsError(error);
    } finally { busy = false; writing = false; }
  }

  async function rotateLink() {
    modalError = '';
    busy = true;
    try {
      await persist();
      if (application.isDirty()) {
        modalError = 'Wait until this page is saved before creating a new link.';
        return;
      }
      writing = true;
      const updated = await rotateSessionLink(sessionServices, currentSession(), document);
      const rotated = { ...updated, epoch: application.nextEpoch() };
      application.send({ type: 'CREATED', session: rotated });
      runtime.navigation.replace(updated.locator);
      modal = 'link';
    } catch (error) {
      handleSettingsError(error);
    } finally { busy = false; writing = false; }
  }

  function handleSettingsError(error: unknown) {
    if (error instanceof ApplicationError && error.code === 'conflict') {
      application.send({ type: 'SAVE.FAILED', status: 'conflict' });
      modal = null;
    } else if (error instanceof ApplicationError && ['unauthorized', 'forbidden', 'not-found'].includes(error.code)) {
      application.send({ type: 'SAVE.FAILED', status: 'revoked' });
      modal = null;
    } else modalError = 'Unable to update this page. Try again.';
  }

  function openModal(value: Modal) {
    settingsOpen = false;
    modalError = '';
    modal = value;
  }

  function currentSession(): OpenSession {
    const current = application.currentSession();
    if (!current) throw new Error('No open session');
    return current;
  }

  function prepareNavigation() {
    password = '';
    passwordRepeat = '';
    unlockPassword = '';
    authError = '';
    focusedId = null;
    selectedIds = new Set();
    clearDragState();
    clearHistory();
    query = '';
    modal = null;
    settingsOpen = false;
  }

  function syncLifecycleView(view: AppViewState) {
    if (view.kind === 'start') {
      busy = false;
      void refreshNativeLocators();
      return;
    }
    if (view.kind === 'loading') {
      prepareNavigation();
      return;
    }
    if (view.kind === 'locked') {
      busy = false;
      return;
    }
    if (view.kind === 'missing') {
      busy = false;
      return;
    }
    busy = false;
  }
  return {
    get screen() { return screen; },
    get startMode() { return startMode; },
    set startMode(value) { startMode = value; },
    get theme() { return theme; },
    set theme(value) { theme = value; },
    get busy() { return busy; },
    set busy(value) { busy = value; },
    get password() { return password; },
    set password(value) { password = value; },
    get passwordRepeat() { return passwordRepeat; },
    set passwordRepeat(value) { passwordRepeat = value; },
    get pageLinkInput() { return pageLinkInput; },
    set pageLinkInput(value) { pageLinkInput = value; },
    get unlockPassword() { return unlockPassword; },
    set unlockPassword(value) { unlockPassword = value; },
    get authError() { return authError; },
    set authError(value) { authError = value; },
    get pageLinkError() { return pageLinkError; },
    set pageLinkError(value) { pageLinkError = value; },
    get localSessionError() { return localSessionError; },
    set localSessionError(value) { localSessionError = value; },
    get nativeLocators() { return nativeLocators; },
    set nativeLocators(value) { nativeLocators = value; },
    get document() { return document; },
    get query() { return query; },
    set query(value) { query = value; },
    get searchOpen() { return searchOpen; },
    set searchOpen(value) { searchOpen = value; },
    get saveState() { return saveState; },
    get settingsOpen() { return settingsOpen; },
    set settingsOpen(value) { settingsOpen = value; },
    get linkCopied() { return linkCopied; },
    set linkCopied(value) { linkCopied = value; },
    get linkCopyError() { return linkCopyError; },
    set linkCopyError(value) { linkCopyError = value; },
    get autosaveStopped() { return autosaveStopped; },
    get focusedId() { return focusedId; },
    set focusedId(value) { focusedId = value; },
    get selectedIds() { return selectedIds; },
    get draggedId() { return draggedId; },
    get dropTarget() { return dropTarget; },
    get importInput() { return importInput; },
    set importInput(value) { importInput = value; },
    get modal() { return modal; },
    set modal(value) { modal = value; },
    get newPassword() { return newPassword; },
    set newPassword(value) { newPassword = value; },
    get newPasswordRepeat() { return newPasswordRepeat; },
    set newPasswordRepeat(value) { newPasswordRepeat = value; },
    get modalError() { return modalError; },
    set modalError(value) { modalError = value; },
    get historyAvailable() { return historyAvailable; },
    get suggestions() { return suggestions; },
    get rootIds() { return rootIds; },
    get displayRootIds() { return displayRootIds; },
    get filteredIds() { return filteredIds; },
    get matchedIds() { return matchedIds; },
    get trail() { return trail; },
    get accessLink() { return accessLink; },
    get accessKind() { return runtime.access.kind; },
    toggleTheme,
    setStartMode,
    createPage,
    openPageLink,
    openLocator,
    returnToStart,
    unlockPage,
    updateQuery,
    handleSearchKeydown,
    returnToEditor,
    chooseSuggestion,
    copyLink,
    exportMarkdownFile,
    chooseMarkdownFile,
    openModal,
    importMarkdownFile,
    reload: () => runtime.navigation.reload(),
    focusBranch,
    addFirstChild,
    register,
    rememberEditor,
    updateText,
    handleBlockKeydown,
    toggle,
    dragStart,
    dragOver,
    drop,
    dragEnd: clearDragState,
    rememberLocalLocator,
    submitPasswordChange,
    rotateLink,
  };
}
