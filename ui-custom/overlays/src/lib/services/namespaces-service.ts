import { get } from 'svelte/store';

import { namespaces } from '$lib/stores/namespaces';
import { getAuthUser } from '$lib/stores/auth-user';
import { toaster } from '$lib/stores/toaster';
import type {
  DescribeNamespaceResponse,
  ListNamespacesResponse,
} from '$lib/types';
import type { Settings } from '$lib/types/global';
import { paginated } from '$lib/utilities/paginated';
import { requestFromAPI } from '$lib/utilities/request-from-api';
import { routeForApi } from '$lib/utilities/route-for-api';
import {
  toNamespaceArchivalStateReadable,
  toNamespaceStateReadable,
} from '$lib/utilities/screaming-enums';

const emptyNamespace = {
  namespaces: [],
};

const toNamespaceDetails = (
  namespace: DescribeNamespaceResponse,
): DescribeNamespaceResponse => {
  if (namespace.config) {
    namespace.config.historyArchivalState = toNamespaceArchivalStateReadable(
      namespace.config.historyArchivalState,
    );
    namespace.config.visibilityArchivalState = toNamespaceArchivalStateReadable(
      namespace.config.visibilityArchivalState,
    );
  }

  if (namespace.namespaceInfo) {
    namespace.namespaceInfo.state = toNamespaceStateReadable(
      namespace.namespaceInfo?.state,
    );
  }
  return namespace;
};

export async function fetchNamespaces(
  settings: Settings,
  request = fetch,
): Promise<void> {
  const { showTemporalSystemNamespace, runtimeEnvironment } = settings;

  if (runtimeEnvironment.isCloud) {
    namespaces.set([]);
    return;
  }

  try {
    if (settings.auth.enabled) {
      const user = getAuthUser();
      if (!user?.accessToken) {
        namespaces.set([]);
        return;
      }
      const userNamespaces = await fetchUserNamespaces(request, user);
      if (userNamespaces !== null) {
        const detailed: DescribeNamespaceResponse[] = [];
        for (const ns of userNamespaces) {
          try {
            const route = routeForApi('namespace', { namespace: ns.name });
            const result = await requestFromAPI<DescribeNamespaceResponse>(route, {
              request,
              onError: () => {},
            });
            if (result) {
              detailed.push(toNamespaceDetails(result));
            }
          } catch {
            // Skip namespaces we can't describe
          }
        }
        namespaces.set(detailed);
        return;
      }
      namespaces.set([]);
      return;
    }

    const route = routeForApi('namespaces');
    const results = await paginated(async (token: string) =>
      requestFromAPI<ListNamespacesResponse>(route, {
        request,
        token,
        onError: () =>
          toaster.push({
            variant: 'error',
            message: 'Unable to fetch namespaces',
          }),
      }),
    );

    const _namespaces: DescribeNamespaceResponse[] = (results?.namespaces ?? [])
      .filter(
        (namespace: DescribeNamespaceResponse) =>
          showTemporalSystemNamespace ||
          namespace.namespaceInfo.name !== 'temporal-system',
      )
      .map(toNamespaceDetails);

    namespaces.set(_namespaces);
  } catch {
    namespaces.set([]);
  }
}

function authHeaders(user: { accessToken?: string; idToken?: string } | null): Record<string, string> {
  if (!user) return {};
  const headers: Record<string, string> = {};
  if (user.accessToken) headers['Authorization'] = `Bearer ${user.accessToken}`;
  if (user.idToken) headers['Authorization-Extras'] = user.idToken;
  return headers;
}

async function fetchUserNamespaces(
  request: typeof fetch,
  user: { accessToken?: string; idToken?: string } | null,
): Promise<Array<{ name: string; type: string; description: string; state: string }> | null> {
  try {
    const res = await request('/api/v1/user/namespaces', {
      method: 'GET',
      headers: authHeaders(user),
      credentials: 'include',
    });

    if (res.ok) {
      const data = await res.json();
      return data.namespaces ?? [];
    }
  } catch (e) {
    console.error('[namespaces-service] fetchUserNamespaces failed:', e);
  }
  return null;
}

export async function createNamespace(
  description?: string,
  request = fetch,
): Promise<{ namespace: string; type: string; description: string } | null> {
  const user = getAuthUser();
  try {
    const body: Record<string, string> = {};
    if (description) {
      body.description = description;
    }

    const res = await request('/api/v1/user/namespaces', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...authHeaders(user),
      },
      credentials: 'include',
      body: JSON.stringify(body),
    });

    if (res.ok || res.status === 201) {
      return await res.json();
    }

    const errData = await res.json().catch(() => ({}));
    toaster.push({
      variant: 'error',
      message: errData.message || 'Failed to create namespace',
    });
  } catch (e) {
    console.error('[namespaces-service] createNamespace failed:', e);
    toaster.push({
      variant: 'error',
      message: 'Failed to create namespace',
    });
  }
  return null;
}

export async function fetchNamespace(
  namespace: string,
  settings?: Settings,
  request = fetch,
): Promise<DescribeNamespaceResponse> {
  const [empty] = emptyNamespace.namespaces;

  if (settings?.runtimeEnvironment?.isCloud) {
    return empty;
  }

  const route = routeForApi('namespace', { namespace });
  const results = await requestFromAPI<DescribeNamespaceResponse>(route, {
    request,
    onError: () =>
      toaster.push({ variant: 'error', message: 'Unable to fetch namespace' }),
  });

  return results ? toNamespaceDetails(results) : empty;
}
