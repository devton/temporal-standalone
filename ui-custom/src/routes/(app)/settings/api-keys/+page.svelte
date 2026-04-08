<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import Button from '$lib/holocene/button.svelte';
  import Icon from '$lib/holocene/icon/icon.svelte';
  import { authUser } from '$lib/stores/auth-user';
  import { translate } from '$lib/i18n/translate';

  interface APIKey {
    id: string;
    name: string;
    description?: string;
    scopes: string[];
    createdAt: string;
    lastUsedAt?: string;
    expiresAt: string;
    createdBy: string;
  }

  interface CreateResponse {
    id: string;
    name: string;
    description?: string;
    scopes: string[];
    createdAt: string;
    expiresAt: string;
    createdBy: string;
    token: string;
  }

  let apiKeys = $state<APIKey[]>([]);
  let loading = $state(true);
  let error = $state('');
  let showCreateModal = $state(false);
  let newKeyName = $state('');
  let newKeyDescription = $state('');
  let newKeyScopes = $state('default');
  let newKeyExpiry = $state('1y');
  let creating = $state(false);
  let createdToken = $state('');
  let showTokenModal = $state(false);
  let copySuccess = $state(false);

  onMount(async () => {
    await loadAPIKeys();
  });

  async function loadAPIKeys() {
    loading = true;
    error = '';
    
    try {
      const response = await fetch('/api/v1/api-keys', {
        headers: {
          'Authorization': `Bearer ${$authUser.accessToken}`
        }
      });

      if (!response.ok) {
        if (response.status === 401) {
          error = 'Authentication required. Please log in.';
        } else {
          error = `Failed to load API keys: ${response.statusText}`;
        }
        return;
      }

      apiKeys = await response.json();
    } catch (e) {
      error = `Failed to load API keys: ${e}`;
    } finally {
      loading = false;
    }
  }

  async function createKey() {
    if (!newKeyName.trim()) {
      error = 'Please enter a name for the API key';
      return;
    }

    creating = true;
    error = '';

    const scopes = newKeyScopes === 'all' 
      ? ['*'] 
      : newKeyScopes.split(',').map(s => s.trim()).filter(s => s);

    try {
      const response = await fetch('/api/v1/api-keys', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${$authUser.accessToken}`
        },
        body: JSON.stringify({
          name: newKeyName,
          description: newKeyDescription,
          scopes: scopes,
          expiresIn: newKeyExpiry
        })
      });

      if (!response.ok) {
        error = `Failed to create API key: ${response.statusText}`;
        return;
      }

      const data: CreateResponse = await response.json();
      createdToken = data.token;
      showCreateModal = false;
      showTokenModal = true;
      
      // Reset form
      newKeyName = '';
      newKeyDescription = '';
      newKeyScopes = 'default';
      newKeyExpiry = '1y';

      // Reload list
      await loadAPIKeys();
    } catch (e) {
      error = `Failed to create API key: ${e}`;
    } finally {
      creating = false;
    }
  }

  async function deleteKey(id: string) {
    if (!confirm('Are you sure you want to delete this API key? This action cannot be undone.')) {
      return;
    }

    try {
      const response = await fetch(`/api/v1/api-keys/${id}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${$authUser.accessToken}`
        }
      });

      if (!response.ok) {
        error = `Failed to delete API key: ${response.statusText}`;
        return;
      }

      await loadAPIKeys();
    } catch (e) {
      error = `Failed to delete API key: ${e}`;
    }
  }

  async function copyToken() {
    try {
      await navigator.clipboard.writeText(createdToken);
      copySuccess = true;
      setTimeout(() => {
        copySuccess = false;
      }, 2000);
    } catch (e) {
      console.error('Failed to copy:', e);
    }
  }

  function formatDate(dateStr: string): string {
    if (!dateStr) return 'N/A';
    const date = new Date(dateStr);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
  }

  function closeTokenModal() {
    showTokenModal = false;
    createdToken = '';
  }
</script>

<div class="flex flex-col gap-6">
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-2xl font-semibold">{translate('api-keys.title') || 'API Keys'}</h1>
      <p class="text-subtle mt-1">
        {translate('api-keys.description') || 'Generate tokens for SDK authentication'}
      </p>
    </div>
    <Button onclick={() => showCreateModal = true}>
      <Icon name="plus" />
      {translate('api-keys.generate') || 'Generate New Key'}
    </Button>
  </div>

  {#if error}
    <div class="rounded-lg bg-red-500/10 p-4 text-red-500">
      {error}
    </div>
  {/if}

  {#if loading}
    <div class="flex items-center justify-center p-8">
      <Icon name="loading" class="animate-spin" />
      <span class="ml-2">{translate('common.loading') || 'Loading...'}</span>
    </div>
  {:else if apiKeys.length === 0}
    <div class="flex flex-col items-center justify-center gap-4 p-12 text-center">
      <Icon name="key" class="text-4xl text-subtle" />
      <div>
        <h3 class="text-lg font-medium">{translate('api-keys.no-keys') || 'No API Keys'}</h3>
        <p class="text-subtle mt-1">
          {translate('api-keys.no-keys-desc') || 'Generate your first API key to use with SDKs'}
        </p>
      </div>
      <Button onclick={() => showCreateModal = true}>
        <Icon name="plus" />
        {translate('api-keys.generate') || 'Generate New Key'}
      </Button>
    </div>
  {:else}
    <div class="flex flex-col gap-4">
      {#each apiKeys as key (key.id)}
        <div class="rounded-lg border border-base-600 bg-base-800 p-4">
          <div class="flex items-start justify-between">
            <div>
              <h3 class="font-medium">{key.name}</h3>
              {#if key.description}
                <p class="text-sm text-subtle mt-1">{key.description}</p>
              {/if}
            </div>
            <Button variant="ghost" size="sm" onclick={() => deleteKey(key.id)}>
              <Icon name="trash" />
            </Button>
          </div>
          <div class="mt-3 flex flex-wrap gap-2">
            {#each key.scopes as scope}
              <span class="rounded bg-base-700 px-2 py-0.5 text-xs">{scope}</span>
            {/each}
          </div>
          <div class="mt-3 flex gap-6 text-sm text-subtle">
            <div>
              <span class="font-medium">Created:</span> {formatDate(key.createdAt)}
            </div>
            <div>
              <span class="font-medium">Expires:</span> {formatDate(key.expiresAt)}
            </div>
            {#if key.lastUsedAt}
              <div>
                <span class="font-medium">Last used:</span> {formatDate(key.lastUsedAt)}
              </div>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Create Modal -->
{#if showCreateModal}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onclick={() => showCreateModal = false}>
    <div class="w-full max-w-md rounded-lg bg-base-800 p-6 shadow-xl" onclick={(e) => e.stopPropagation()}>
      <h2 class="text-xl font-semibold">{translate('api-keys.create') || 'Create API Key'}</h2>
      
      <div class="mt-4 flex flex-col gap-4">
        <div>
          <label class="block text-sm font-medium mb-1">
            {translate('api-keys.name') || 'Name'} *
          </label>
          <input
            type="text"
            bind:value={newKeyName}
            class="w-full rounded border border-base-600 bg-base-900 px-3 py-2"
            placeholder="e.g., Production Worker"
          />
        </div>

        <div>
          <label class="block text-sm font-medium mb-1">
            {translate('api-keys.description') || 'Description'}
          </label>
          <input
            type="text"
            bind:value={newKeyDescription}
            class="w-full rounded border border-base-600 bg-base-900 px-3 py-2"
            placeholder="e.g., Worker for finance namespace"
          />
        </div>

        <div>
          <label class="block text-sm font-medium mb-1">
            {translate('api-keys.scopes') || 'Scopes'}
          </label>
          <select
            bind:value={newKeyScopes}
            class="w-full rounded border border-base-600 bg-base-900 px-3 py-2"
          >
            <option value="default">{translate('api-keys.default-namespaces') || 'Default namespaces'}</option>
            <option value="all">{translate('api-keys.all-namespaces') || 'All namespaces'}</option>
          </select>
          <p class="mt-1 text-xs text-subtle">
            {translate('api-keys.scopes-hint') || 'Controls which namespaces this key can access'}
          </p>
        </div>

        <div>
          <label class="block text-sm font-medium mb-1">
            {translate('api-keys.expiry') || 'Expires'}
          </label>
          <select
            bind:value={newKeyExpiry}
            class="w-full rounded border border-base-600 bg-base-900 px-3 py-2"
          >
            <option value="30d">{translate('api-keys.30-days') || '30 days'}</option>
            <option value="90d">{translate('api-keys.90-days') || '90 days'}</option>
            <option value="1y" selected>{translate('api-keys.1-year') || '1 year'}</option>
            <option value="never">{translate('api-keys.never') || 'Never (10 years)'}</option>
          </select>
        </div>
      </div>

      {#if error}
        <div class="mt-4 rounded bg-red-500/10 p-2 text-sm text-red-500">
          {error}
        </div>
      {/if}

      <div class="mt-6 flex justify-end gap-3">
        <Button variant="ghost" onclick={() => showCreateModal = false}>
          {translate('common.cancel') || 'Cancel'}
        </Button>
        <Button onclick={createKey} disabled={creating}>
          {#if creating}
            <Icon name="loading" class="animate-spin" />
          {/if}
          {translate('api-keys.create') || 'Create'}
        </Button>
      </div>
    </div>
  </div>
{/if}

<!-- Token Modal (shown once after creation) -->
{#if showTokenModal}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
    <div class="w-full max-w-lg rounded-lg bg-base-800 p-6 shadow-xl">
      <h2 class="text-xl font-semibold text-green-400">
        {translate('api-keys.created') || 'API Key Created!'}
      </h2>
      
      <p class="mt-2 text-subtle">
        {translate('api-keys.token-warning') || 'Copy this token now. You won\'t be able to see it again!'}
      </p>

      <div class="mt-4">
        <textarea
          readonly
          class="w-full rounded border border-base-600 bg-base-900 p-3 font-mono text-sm"
          rows="6"
        >{createdToken}</textarea>
      </div>

      <div class="mt-4 flex justify-end gap-3">
        <Button onclick={copyToken}>
          {#if copySuccess}
            <Icon name="check" />
            {translate('common.copied') || 'Copied!'}
          {:else}
            <Icon name="copy" />
            {translate('common.copy') || 'Copy'}
          {/if}
        </Button>
        <Button onclick={closeTokenModal}>
          {translate('common.done') || 'Done'}
        </Button>
      </div>
    </div>
  </div>
{/if}
