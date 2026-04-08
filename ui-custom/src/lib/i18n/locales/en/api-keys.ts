export const Namespace = 'api-keys';

export const Strings = {
  title: 'API Keys',
  description: 'Generate tokens for SDK authentication',
  generate: 'Generate New Key',
  create: 'Create API Key',
  no_keys: 'No API Keys',
  no_keys_desc: 'Generate your first API key to use with SDKs',
  name: 'Name',
  'description': 'Description',
  scopes: 'Scopes',
  scopes_hint: 'Controls which namespaces this key can access',
  'default-namespaces': 'Default namespaces',
  'all-namespaces': 'All namespaces',
  expiry: 'Expires',
  '30-days': '30 days',
  '90-days': '90 days',
  '1-year': '1 year',
  never: 'Never (10 years)',
  created: 'API Key Created!',
  token_warning: 'Copy this token now. You won\'t be able to see it again!',
  delete_confirm: 'Are you sure you want to delete this API key? This action cannot be undone.',
} as const;
