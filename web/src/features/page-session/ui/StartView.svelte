<script lang="ts">
  import { pageIDFromLocator } from '../logic/page-link';
  import { minimumPasswordLength, passwordPattern } from '../logic/password';
  import type { Screen, StartMode } from '../logic/view-model';

  export let screen: Screen;
  export let startMode: StartMode;
  export let theme: 'light' | 'dark';
  export let busy: boolean;
  export let password: string;
  export let passwordRepeat: string;
  export let pageLinkInput: string;
  export let unlockPassword: string;
  export let authError: string;
  export let pageLinkError: string;
  export let localSessionError: string;
  export let historyAvailable: boolean;
  export let locators: string[];
  export let toggleTheme: () => void;
  export let setStartMode: (mode: StartMode) => void;
  export let createPage: () => void | Promise<void>;
  export let openPageLink: () => void | Promise<void>;
  export let openLocator: (locator: string) => void | Promise<void>;
  export let returnToStart: () => void | Promise<void>;
  export let unlockPage: () => void | Promise<void>;
</script>

<header class="topbar auth-topbar">
  <div class="wordmark"><span class="brand-mark">S</span><span>singlepage / outline</span></div>
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
    {#if screen === 'create'}
      <div class="auth-card start-card">
        <div class="start-toggle" role="tablist" aria-label="Start action">
          <button id="start-create-tab" type="button" role="tab" aria-selected={startMode === 'create'} aria-controls="start-create-panel" class:active={startMode === 'create'} on:click={() => setStartMode('create')}>Create</button>
          <button id="start-open-tab" type="button" role="tab" aria-selected={startMode === 'open'} aria-controls="start-open-panel" class:active={startMode === 'open'} on:click={() => setStartMode('open')}>Open</button>
        </div>

        {#if startMode === 'create'}
          <div id="start-create-panel" role="tabpanel" aria-labelledby="start-create-tab">
            <form on:submit|preventDefault={createPage}>
              <div class="eyebrow">New page</div>
              <h2>Create your page</h2>
              <p>Choose a password with at least 8 characters, including a letter and a number, and keep the link you receive.</p>
              <label class="field"><span>Password</span><input type="password" autocomplete="new-password" minlength={minimumPasswordLength} pattern={passwordPattern} title="Use at least 8 characters, including a letter and a number." bind:value={password} required /></label>
              <label class="field"><span>Repeat password</span><input type="password" autocomplete="new-password" minlength={minimumPasswordLength} pattern={passwordPattern} title="Use at least 8 characters, including a letter and a number." bind:value={passwordRepeat} required /></label>
              <button class="primary" disabled={busy}>{busy ? 'Creating…' : 'Create page'}</button>
              {#if authError}<p class="error" role="alert">{authError}</p>{/if}
            </form>
          </div>
        {:else}
          <div id="start-open-panel" role="tabpanel" aria-labelledby="start-open-tab">
            <form on:submit|preventDefault={openPageLink}>
              <div class="eyebrow">Existing page</div>
              <h2>Open a page</h2>
              <label class="field"><span>Page link</span><input type="text" autocomplete="url" placeholder="/p/…#… or https://…" bind:value={pageLinkInput} required /></label>
              <button class="secondary start-open" disabled={busy || !pageLinkInput.trim()}>Open link</button>
              {#if pageLinkError}<p class="error" role="alert">{pageLinkError}</p>{/if}
              {#if historyAvailable && locators.length}
                <div class="page-history">
                  <div class="page-history-title">Previously opened</div>
                  {#each locators as locator}
                    <button type="button" disabled={busy} on:click={() => openLocator(locator)}>
                      <span>Page {pageIDFromLocator(locator)}</span><span aria-hidden="true">›</span>
                    </button>
                  {/each}
                </div>
              {/if}
              {#if localSessionError}<p class="error" role="alert">{localSessionError}</p>{/if}
            </form>
          </div>
        {/if}
      </div>
    {:else}
      <form class="auth-card" on:submit|preventDefault={unlockPage}>
        <div class="eyebrow">Welcome back</div>
        {#if screen === 'loading'}
          <h2>Opening page…</h2>
        {:else if screen === 'missing'}
          <h2>Page not found</h2><p>Check the link and try again.</p>
          <button class="secondary" type="button" on:click={returnToStart}>Back to start</button>
        {:else}
          <h2>Unlock page</h2>
          <p>Enter the password for this page.</p>
          <label class="field"><span>Password</span><input type="password" autocomplete="current-password" bind:value={unlockPassword} required /></label>
          <button class="primary" disabled={busy}>{busy ? 'Opening…' : 'Open'}</button>
          <button class="secondary auth-back" type="button" disabled={busy} on:click={returnToStart}>Back to start</button>
        {/if}
        {#if authError}<p class="error" role="alert">{authError}</p>{/if}
        {#if localSessionError}<p class="error" role="alert">{localSessionError}</p>{/if}
      </form>
    {/if}
  </section>
</main>
