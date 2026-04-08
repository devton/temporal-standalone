<script lang="ts">
  import { onMount } from 'svelte';
  import PageTitle from '$lib/components/page-title.svelte';
  import Panel from '$lib/components/panel.svelte';

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

  let keys: APIKey[] = [];
  let loading = true;
  let error = '';
  let showCreateModal = false;
  let newKeyName = '';
  let newKeyDescription = '';
  let createdKey: APIKey | null = null;
  let copying = false;

  async function loadKeys() {
    try {
      loading = true;
      const response = await fetch('/api/v1/api-keys');
      if (!response.ok) throw new Error('Failed to load API keys');
      const data = await response.json();
      keys = data.keys || [];
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load API keys';
    } finally {
      loading = false;
    }
  }

  async function createKey() {
    try {
      const response = await fetch('/api/v1/api-keys', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: newKeyName,
          description: newKeyDescription,
        }),
      });
      if (!response.ok) throw new Error('Failed to create API key');
      createdKey = await response.json();
      showCreateModal = false;
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
      const response = await fetch(`/api/v1/api-keys/${id}`, { method: 'DELETE' });
      if (!response.ok) throw new Error('Failed to delete API key');
      await loadKeys();
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to delete API key';
    }
  }

  async function copyToClipboard(text: string) {
    copying = true;
    await navigator.clipboard.writeText(text);
    setTimeout(() => (copying = false), 2000);
  }

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleString();
  }

  onMount(loadKeys);
</script>

<svelte:head>
  <title>API Keys - Temporal</title>
</svelte:head>

<PageTitle>API Keys</PageTitle>

{#if error}
  <div class="error-banner">{error}</div>
{/if}

{#if createdKey}
  <Panel>
    <h2>API Key Created</h2>
    <p>Your new API key has been created. Copy the token below - it will only be shown once:</p>
    <div class="token-display">
      <code>{createdKey.keySecret}</code>
      <button on:click={() => copyToClipboard(createdKey!.keySecret || '')}>
        {copying ? 'Copied!' : 'Copy'}
      </button>
    </div>
    <button on:click={() => (createdKey = null)}>Close</button>
  </Panel>
{/if}

<Panel>
  <div class="header-row">
    <h2>Your API Keys</h2>
    <button on:click={() => (showCreateModal = true)}>Create API Key</button>
  </div>

  {#if loading}
    <p>Loading...</p>
  {:else if keys.length === 0}
    <p>No API keys yet. Create one to get started.</p>
  {:else}
    <table>
      <thead>
        <tr>
          <th>Name</th>
          <th>Key ID</th>
          <th>Created</th>
          <th>Last Used</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        {#each keys as key}
          <tr>
            <td>{key.name}</td>
            <td><code>{key.keyId}</code></td>
            <td>{formatDate(key.createdAt)}</td>
            <td>{key.lastUsedAt ? formatDate(key.lastUsedAt) : 'Never'}</td>
            <td>
              <button on:click={() => deleteKey(key.id)} class="delete-btn">Delete</button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</Panel>

{#if showCreateModal}
  <div class="modal-overlay">
    <div class="modal">
      <h2>Create New API Key</h2>
      <label>
        Name
        <input type="text" bind:value={newKeyName} placeholder="My API Key" />
      </label>
      <label>
        Description (optional)
        <input type="text" bind:value={newKeyDescription} placeholder="Used for..." />
      </label>
      <div class="modal-actions">
        <button on:click={() => (showCreateModal = false)}>Cancel</button>
        <button on:click={createKey} disabled={!newKeyName}>Create</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .header-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .error-banner {
    background: #fee;
    color: #c00;
    padding: 1rem;
    border-radius: 4px;
    margin-bottom: 1rem;
  }

  .token-display {
    display: flex;
    gap: 0.5rem;
    margin: 1rem 0;
  }

  .token-display code {
    flex: 1;
    padding: 0.5rem;
    background: #f5f5f5;
    border-radius: 4px;
    word-break: break-all;
  }

  table {
    width: 100%;
    border-collapse: collapse;
  }

  th, td {
    padding: 0.75rem;
    text-align: left;
    border-bottom: 1px solid #eee;
  }

  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .modal {
    background: white;
    padding: 2rem;
    border-radius: 8px;
    min-width: 400px;
  }

  .modal label {
    display: block;
    margin-bottom: 1rem;
  }

  .modal input {
    width: 100%;
    padding: 0.5rem;
    margin-top: 0.25rem;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 1rem;
  }

  .delete-btn {
    color: #c00;
  }
</style>
