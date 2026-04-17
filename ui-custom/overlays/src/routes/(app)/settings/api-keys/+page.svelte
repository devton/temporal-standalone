<script lang="ts">
  import { onMount } from 'svelte';
  import { getAuthUser } from '$lib/stores/auth-user';
  import PageTitle from '$lib/components/page-title.svelte';
  import Panel from '$lib/components/panel.svelte';
  import Button from '$lib/holocene/button.svelte';
  import Input from '$lib/holocene/input/input.svelte';

  interface APIKey {
    id: string;
    name: string;
    description?: string;
    keyId: string;
    keySecret?: string;
    createdAt: string;
    expiresAt?: string;
    ownerId: string;
    lastUsedAt?: string;
  }

  let keys: APIKey[] = $state([]);
  let loading = $state(true);
  let error = $state('');
  let showModal = $state(false);
  let newKeyName = $state('');
  let newKeyDescription = $state('');
  let createdKey: APIKey | null = $state(null);
  let copying = $state(false);

  function getHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'Caller-Type': 'operator',
    };

    const user = getAuthUser();
    if (user?.accessToken) {
      headers['Authorization'] = `Bearer ${user.accessToken}`;
    }
    if (user?.idToken) {
      headers['Authorization-Extras'] = user.idToken;
    }

    try {
      const cookies = document.cookie.split(';');
      const csrfCookie = cookies.find((c) => c.includes('_csrf='));
      if (csrfCookie) {
        headers['X-CSRF-TOKEN'] = csrfCookie.trim().slice('_csrf='.length);
      }
    } catch {}

    return headers;
  }

  async function apiFetch<T>(
    url: string,
    options: RequestInit = {},
  ): Promise<T> {
    const res = await fetch(url, {
      ...options,
      headers: { ...getHeaders(), ...(options.headers || {}) },
      credentials: 'include',
    });

    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || `Request failed: ${res.status}`);
    }

    if (res.status === 204) return {} as T;
    return res.json();
  }

  async function loadKeys() {
    try {
      loading = true;
      const data = await apiFetch<{ keys: APIKey[] }>('/api/v1/api-keys');
      keys = data.keys || [];
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load API keys';
    } finally {
      loading = false;
    }
  }

  async function createKey() {
    try {
      error = '';
      const key = await apiFetch<APIKey>('/api/v1/api-keys', {
        method: 'POST',
        body: JSON.stringify({
          name: newKeyName,
          description: newKeyDescription,
        }),
      });
      createdKey = key;
      showModal = false;
      newKeyName = '';
      newKeyDescription = '';
      await loadKeys();
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to create API key';
    }
  }

  async function deleteKey(id: string) {
    if (!confirm('Are you sure you want to delete this API key?')) return;
    try {
      await apiFetch<void>(`/api/v1/api-keys/${id}`, { method: 'DELETE' });
      await loadKeys();
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to delete API key';
    }
  }

  async function copyToken(text: string) {
    copying = true;
    await navigator.clipboard.writeText(text);
    setTimeout(() => (copying = false), 2000);
  }

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleString();
  }

  function openModal() {
    showModal = true;
    newKeyName = '';
    newKeyDescription = '';
    error = '';
  }

  function closeModal() {
    showModal = false;
  }

  function dismissCreatedKey() {
    createdKey = null;
  }

  onMount(loadKeys);
</script>

<svelte:head>
  <title>API Keys - Temporal</title>
</svelte:head>

<PageTitle>API Keys</PageTitle>

{#if error}
  <div class="mb-4 rounded bg-red-500/10 p-4 text-sm text-red-500">{error}</div>
{/if}

{#if createdKey}
  <Panel>
    <h2 class="text-xl font-semibold text-primary">API Key Created</h2>
    <p class="mt-2 text-sm text-subtle">
      Copy the token below — it will only be shown once.
    </p>
    <div class="mt-4 flex items-center gap-2">
      <code
        class="flex-1 break-all rounded border border-secondary bg-surface-secondary px-3 py-2 text-sm text-primary"
      >
        {createdKey.keySecret}
      </code>
      <Button
        variant="secondary"
        size="sm"
        on:click={() => copyToken(createdKey!.keySecret || '')}
      >
        {copying ? 'Copied!' : 'Copy'}
      </Button>
    </div>
    <div class="mt-4">
      <Button variant="ghost" on:click={dismissCreatedKey}>Close</Button>
    </div>
  </Panel>
{/if}

<Panel>
  <div class="mb-4 flex items-center justify-between">
    <h2 class="text-xl font-semibold text-primary">Your API Keys</h2>
    <Button on:click={openModal}>
      + Create API Key
    </Button>
  </div>

  {#if loading}
    <p class="text-subtle">Loading...</p>
  {:else if keys.length === 0}
    <p class="text-subtle">No API keys yet. Create one to get started.</p>
  {:else}
    <table class="w-full border-collapse">
      <thead>
        <tr class="border-b border-secondary text-left text-sm text-subtle">
          <th class="px-3 py-2 font-medium">Name</th>
          <th class="px-3 py-2 font-medium">Key ID</th>
          <th class="px-3 py-2 font-medium">Description</th>
          <th class="px-3 py-2 font-medium">Created</th>
          <th class="px-3 py-2 font-medium">Last Used</th>
          <th class="px-3 py-2 font-medium">Actions</th>
        </tr>
      </thead>
      <tbody>
        {#each keys as key}
          <tr class="border-b border-secondary text-sm text-primary">
            <td class="px-3 py-3">{key.name}</td>
            <td class="px-3 py-3 font-mono text-xs">{key.keyId}</td>
            <td class="px-3 py-3">{key.description || '—'}</td>
            <td class="px-3 py-3">{formatDate(key.createdAt)}</td>
            <td class="px-3 py-3">
              {key.lastUsedAt ? formatDate(key.lastUsedAt) : 'Never'}
            </td>
            <td class="px-3 py-3">
              <Button
                variant="ghost"
                size="sm"
                on:click={() => deleteKey(key.id)}
              >
                Delete
              </Button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</Panel>

{#if showModal}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    on:click={closeModal}
    on:keydown={() => {}}
  >
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="w-full max-w-lg rounded-lg border border-secondary bg-surface-primary p-6 shadow-xl"
      on:click={(e) => e.stopPropagation()}
      on:keydown={() => {}}
    >
      <div class="mb-4 flex items-center justify-between">
        <h2 class="text-lg font-semibold text-primary">Create New API Key</h2>
        <button
          class="flex h-8 w-8 items-center justify-center rounded-full text-subtle hover:bg-surface-secondary hover:text-primary"
          on:click={closeModal}
        >
          ✕
        </button>
      </div>

      <div class="mb-4">
        <Input
          id="api-key-name"
          label="Name"
          placeholder="My API Key"
          bind:value={newKeyName}
          required={true}
        />
      </div>

      <div class="mb-4">
        <Input
          id="api-key-description"
          label="Description"
          placeholder="Used for..."
          bind:value={newKeyDescription}
        />
      </div>

      <div class="flex justify-end gap-2">
        <Button variant="ghost" on:click={closeModal}>Cancel</Button>
        <Button on:click={createKey} disabled={!newKeyName}>Create</Button>
      </div>
    </div>
  </div>
{/if}
