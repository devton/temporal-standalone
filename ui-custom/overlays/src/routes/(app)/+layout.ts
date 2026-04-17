import { redirect } from '@sveltejs/kit';

import type { LayoutData, LayoutLoad } from './$types';

import { fetchCluster, fetchSystemInfo } from '$lib/services/cluster-service';
import { fetchNamespaces } from '$lib/services/namespaces-service';
import { fetchSettings } from '$lib/services/settings-service';
import { clearAuthUser, getAuthUser, setAuthUser } from '$lib/stores/auth-user';
import type { GetClusterInfoResponse, GetSystemInfoResponse } from '$lib/types';
import type { Settings } from '$lib/types/global';
import {
  cleanAuthUserCookie,
  getAuthUserCookie,
} from '$lib/utilities/auth-user-cookie';
import { isAuthorized } from '$lib/utilities/is-authorized';
import { routeForLoginPage } from '$lib/utilities/route-for';

import '../../app.css';

async function ensureUserNamespace(
  settings: Settings,
  request: typeof fetch,
): Promise<string | null> {
  const user = getAuthUser();
  if (!user?.accessToken) return null;

  try {
    const res = await request('/api/v1/user/ensure-namespace', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${user.accessToken}`,
        ...(user.idToken
          ? { 'Authorization-Extras': user.idToken }
          : {}),
      },
      credentials: 'include',
    });

    if (res.ok) {
      const data = await res.json();
      return data.namespace as string;
    }
  } catch (e) {
    console.error('[Layout] ensure-namespace failed:', e);
  }
  return null;
}

export const load: LayoutLoad = async function ({
  fetch,
}): Promise<LayoutData> {
  const settings: Settings = await fetchSettings(fetch);

  if (!settings.auth.enabled) {
    cleanAuthUserCookie();
    clearAuthUser();
  }

  const authUser = getAuthUserCookie();
  if (authUser?.accessToken) {
    setAuthUser(authUser);
    cleanAuthUserCookie();
  }

  const user = getAuthUser();

  if (!isAuthorized(settings, user)) {
    redirect(302, routeForLoginPage());
  }

  // Auto-create user namespace if auth is enabled
  if (settings.auth.enabled && user?.accessToken) {
    const userNamespace = await ensureUserNamespace(settings, fetch);
    if (userNamespace) {
      try {
        const { lastUsedNamespace } = await import(
          '$lib/stores/namespaces'
        );
        const { get } = await import('svelte/store');
        const current = get(lastUsedNamespace);
        // Only set if not already set to user namespace
        if (!current || current === 'default') {
          lastUsedNamespace.set(userNamespace);
        }
      } catch {
        // Store import failed, continue anyway
      }
    }
  }

  fetchNamespaces(settings, fetch);

  const cluster: GetClusterInfoResponse = await fetchCluster(settings, fetch);
  const systemInfo: GetSystemInfoResponse = await fetchSystemInfo(
    settings,
    fetch,
  );

  return {
    user,
    settings,
    cluster,
    systemInfo,
  };
};
