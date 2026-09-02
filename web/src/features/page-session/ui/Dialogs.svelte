<script lang="ts">
  import { minimumPasswordLength, passwordPattern } from '../logic/password';
  import type { Modal } from '../logic/view-model';

  export let modal: Modal;
  export let accessKind: 'browser' | 'local-vault';
  export let localSessionError: string;
  export let accessLink: string;
  export let linkCopyError: string;
  export let linkCopied: boolean;
  export let newPassword: string;
  export let newPasswordRepeat: string;
  export let modalError: string;
  export let busy: boolean;
  export let rememberLocalLocator: () => void | Promise<boolean>;
  export let copyLink: () => void | Promise<void>;
  export let submitPasswordChange: () => void | Promise<void>;
  export let rotateLink: () => void | Promise<void>;
</script>

{#if modal === 'link'}
  <div class="modal-backdrop" role="presentation">
    <div class="modal" role="dialog" aria-modal="true" aria-labelledby="link-title">
      <div class="eyebrow">{accessKind === 'local-vault' ? 'Local vault' : 'Access link'}</div>
      <h2 id="link-title">{accessKind === 'local-vault' ? 'Stored on this device.' : 'Save this link.'}</h2>
      {#if accessKind === 'local-vault'}
        {#if localSessionError}
          <p class="error">{localSessionError}</p>
          <div class="button-row"><button class="primary" type="button" on:click={rememberLocalLocator}>Retry</button><button class="secondary" type="button" on:click={() => modal = null}>Done</button></div>
        {:else}
          <p>The app will reopen this vault after a restart. Your password is still required.</p>
          <p class="secret-link">{accessLink}</p>
          {#if linkCopyError}<p class="error" role="alert">{linkCopyError}</p>{/if}
          <div class="button-row"><button class="primary" type="button" on:click={copyLink}>{linkCopied ? 'Copied' : 'Copy link'}</button><button class="secondary" type="button" on:click={() => modal = null}>Done</button></div>
        {/if}
      {:else}
        <p class="secret-link">{accessLink}</p>
        <p>You will also need your password.</p>
        {#if linkCopyError}<p class="error" role="alert">{linkCopyError}</p>{/if}
        <div class="button-row"><button class="primary" type="button" on:click={copyLink}>Copy link</button><button class="secondary" type="button" on:click={() => modal = null}>Done</button></div>
      {/if}
    </div>
  </div>
{:else if modal === 'password'}
  <div class="modal-backdrop" role="presentation">
    <div class="modal" role="dialog" aria-modal="true" aria-labelledby="password-title">
      <form on:submit|preventDefault={submitPasswordChange}>
        <div class="eyebrow">Password</div><h2 id="password-title">Change password</h2><p>Use at least 8 characters, including a letter and a number.</p>
        <label class="field"><span>New password</span><input type="password" autocomplete="new-password" minlength={minimumPasswordLength} pattern={passwordPattern} title="Use at least 8 characters, including a letter and a number." bind:value={newPassword} required /></label>
        <label class="field"><span>Repeat new password</span><input type="password" autocomplete="new-password" minlength={minimumPasswordLength} pattern={passwordPattern} title="Use at least 8 characters, including a letter and a number." bind:value={newPasswordRepeat} required /></label>
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
