<script lang="ts">
  import StartView from '../features/page-session/ui/StartView.svelte';
  import Dialogs from '../features/page-session/ui/Dialogs.svelte';
  import WorkspaceView from '../features/outline-editor/ui/WorkspaceView.svelte';
  import { createAppController } from './logic/app-controller.svelte';
  import type { ApplicationActorBinding } from './logic/app-actor.svelte';
  import type { ApplicationRuntime } from './logic/runtime';

  interface Props {
    runtime: ApplicationRuntime;
    application: ApplicationActorBinding;
  }

  let { runtime, application }: Props = $props();
  // Runtime and actor bindings are immutable for the lifetime of the app shell.
  // svelte-ignore state_referenced_locally
  const controller = createAppController(runtime, application);
</script>

<div class="shell">
  {#if controller.screen === 'create' || controller.screen === 'unlock' || controller.screen === 'loading' || controller.screen === 'missing'}
    <StartView
      screen={controller.screen}
      bind:startMode={controller.startMode}
      theme={controller.theme}
      busy={controller.busy}
      bind:password={controller.password}
      bind:passwordRepeat={controller.passwordRepeat}
      bind:pageLinkInput={controller.pageLinkInput}
      bind:unlockPassword={controller.unlockPassword}
      authError={controller.authError}
      pageLinkError={controller.pageLinkError}
      localSessionError={controller.localSessionError}
      historyAvailable={controller.historyAvailable}
      locators={controller.nativeLocators}
      toggleTheme={controller.toggleTheme}
      setStartMode={controller.setStartMode}
      createPage={controller.createPage}
      openPageLink={controller.openPageLink}
      openLocator={controller.openLocator}
      returnToStart={controller.returnToStart}
      unlockPage={controller.unlockPage}
    />
  {:else}
    <WorkspaceView
      document={controller.document}
      query={controller.query}
      bind:searchOpen={controller.searchOpen}
      suggestions={controller.suggestions}
      saveState={controller.saveState}
      theme={controller.theme}
      bind:settingsOpen={controller.settingsOpen}
      linkCopied={controller.linkCopied}
      linkCopyError={controller.linkCopyError}
      localSessionError={controller.localSessionError}
      historyAvailable={controller.historyAvailable}
      autosaveStopped={controller.autosaveStopped}
      focusedId={controller.focusedId}
      rootIds={controller.rootIds}
      displayRootIds={controller.displayRootIds}
      filteredIds={controller.filteredIds}
      matchedIds={controller.matchedIds}
      trail={controller.trail}
      selectedIds={controller.selectedIds}
      draggedId={controller.draggedId}
      dropTarget={controller.dropTarget}
      bind:importInput={controller.importInput}
      returnToStart={controller.returnToStart}
      updateQuery={controller.updateQuery}
      handleSearchKeydown={controller.handleSearchKeydown}
      returnToEditor={controller.returnToEditor}
      chooseSuggestion={controller.chooseSuggestion}
      toggleTheme={controller.toggleTheme}
      copyLink={controller.copyLink}
      exportMarkdownFile={controller.exportMarkdownFile}
      chooseMarkdownFile={controller.chooseMarkdownFile}
      openModal={controller.openModal}
      importMarkdownFile={controller.importMarkdownFile}
      reload={controller.reload}
      focusBranch={controller.focusBranch}
      addFirstChild={controller.addFirstChild}
      register={controller.register}
      rememberEditor={controller.rememberEditor}
      updateText={controller.updateText}
      handleBlockKeydown={controller.handleBlockKeydown}
      toggle={controller.toggle}
      dragStart={controller.dragStart}
      dragOver={controller.dragOver}
      drop={controller.drop}
      dragEnd={controller.dragEnd}
    />
  {/if}

  <Dialogs
    bind:modal={controller.modal}
    accessKind={controller.accessKind}
    localSessionError={controller.localSessionError}
    accessLink={controller.accessLink}
    linkCopyError={controller.linkCopyError}
    linkCopied={controller.linkCopied}
    bind:newPassword={controller.newPassword}
    bind:newPasswordRepeat={controller.newPasswordRepeat}
    modalError={controller.modalError}
    busy={controller.busy}
    rememberLocalLocator={controller.rememberLocalLocator}
    copyLink={controller.copyLink}
    submitPasswordChange={controller.submitPasswordChange}
    rotateLink={controller.rotateLink}
  />
</div>
