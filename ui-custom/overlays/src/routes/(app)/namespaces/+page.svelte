<script lang="ts">
  import { page } from '$app/stores';

  import PageTitle from '$lib/components/page-title.svelte';
  import EmptyState from '$lib/holocene/empty-state.svelte';
  import Link from '$lib/holocene/link.svelte';
  import Pagination from '$lib/holocene/pagination.svelte';
  import TableHeaderRow from '$lib/holocene/table/table-header-row.svelte';
  import TableRow from '$lib/holocene/table/table-row.svelte';
  import Table from '$lib/holocene/table/table.svelte';
  import { translate } from '$lib/i18n/translate';
  import { namespaces } from '$lib/stores/namespaces';
  import { routeForNamespace } from '$lib/utilities/route-for';

  // Custom: Button, Modal, Input for create namespace
  import Button from '$lib/holocene/button.svelte';
  import Modal from '$lib/holocene/modal.svelte';
  import Input from '$lib/holocene/input/input.svelte';
  import { createNamespace } from '$lib/services/namespaces-service';
  import { fetchNamespaces } from '$lib/services/namespaces-service';
  import { getAuthUser } from '$lib/stores/auth-user';
  import { fetchSettings } from '$lib/services/settings-service';
  import { toaster } from '$lib/stores/toaster';
  import { goto } from '$app/navigation';

  let showModal = $state(false);
  let description = $state('');
  let creating = $state(false);

  async function handleConfirmModal() {
    creating = true;
    try {
      const result = await createNamespace(description || undefined);
      if (result) {
        toaster.push({
          variant: 'success',
          message: `Namespace ${result.namespace} created!`,
        });
        showModal = false;
        description = '';
        // Refresh namespace list
        const settings = await fetchSettings(fetch);
        await fetchNamespaces(settings, fetch);
        // Navigate to the new namespace
        goto(routeForNamespace({ namespace: result.namespace }));
      }
    } finally {
      creating = false;
    }
  }

  let user = $derived(getAuthUser());
  let showCreate = $derived(!!user);
</script>

<PageTitle title="Namespaces" url={$page.url.href} />
<div class="mb-8 flex items-center justify-between">
  <h1 data-testid="namespace-selector-title">
    {translate('common.namespaces')}
  </h1>
  {#if showCreate}
    <Button variant="primary" size="sm" leadingIcon="plus" on:click={() => (showModal = true)}>
      Create Namespace
    </Button>
  {/if}
</div>

{#if $namespaces?.length > 0}
  <Pagination
    items={$namespaces}
    let:visibleItems
    aria-label={translate('common.namespaces')}
    pageSizeSelectLabel={translate('common.per-page')}
    previousButtonLabel={translate('common.previous')}
    nextButtonLabel={translate('common.next')}
  >
    <Table class="w-full">
      <caption class="sr-only" slot="caption"
        >{translate('common.namespaces')}</caption
      >
      <TableHeaderRow slot="headers">
        <th>{translate('common.name')}</th>
        <th>Description</th>
      </TableHeaderRow>
      {#each visibleItems as namespace (namespace.namespaceInfo.name)}
        <TableRow>
          <td>
            <Link
              href={routeForNamespace({
                namespace: namespace.namespaceInfo.name,
              })}>{namespace.namespaceInfo.name}</Link
            >
          </td>
          <td class="text-sm text-subtle">
            {namespace.namespaceInfo?.description || ''}
          </td>
        </TableRow>
      {/each}
    </Table>
  </Pagination>
{:else}
  <EmptyState
    title={translate('namespaces.namespaces-empty-state-title')}
    content={translate('namespaces.namespaces-empty-state-content')}
  />
{/if}

{#if showCreate}
  <Modal
    id="create-namespace-modal"
    open={showModal}
    confirmText={creating ? 'Creating...' : 'Create'}
    cancelText="Cancel"
    loading={creating}
    confirmDisabled={creating}
    on:cancelModal={() => { showModal = false; description = ''; }}
    on:confirmModal={handleConfirmModal}
  >
    <svelte:fragment slot="title">
      Create New Namespace
    </svelte:fragment>
    <svelte:fragment slot="content">
      <div class="flex flex-col gap-4">
        <p class="text-sm text-subtle">
          A unique namespace name will be automatically generated. You can optionally add a description.
        </p>
        <Input
          id="namespace-description"
          label="Description"
          placeholder="Optional description for this namespace"
          bind:value={description}
        />
      </div>
    </svelte:fragment>
  </Modal>
{/if}
