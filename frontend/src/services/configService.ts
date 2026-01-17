import yaml from 'js-yaml';

interface AppConfig {
  exclude?: {
    accounts?: {
      ids?: string[];
      names?: string[];
    };
    regions?: string[];
  };
}

let cachedConfig: AppConfig | null = null;

export async function loadAppConfig(): Promise<AppConfig> {
  if (cachedConfig) return cachedConfig;

  try {
    const response = await fetch('/config.yaml');
    if (!response.ok) {
      console.warn('Could not load config.yaml, using defaults');
      return {};
    }
    const text = await response.text();
    cachedConfig = yaml.load(text) as AppConfig;
    return cachedConfig || {};
  } catch (error) {
    console.warn('Error loading config.yaml:', error);
    return {};
  }
}

function wildcardToRegex(pattern: string): RegExp {
  const escaped = pattern.replace(/[.+^${}()|[\]\\]/g, '\\$&');
  const withWildcards = escaped.replace(/\*/g, '.*').replace(/\?/g, '.');
  return new RegExp(`^${withWildcards}$`, 'i');
}

export function shouldExcludeAccount(
  account: { id: string; name: string },
  config: AppConfig
): boolean {
  const excludeIds = config.exclude?.accounts?.ids || [];
  const excludeNames = config.exclude?.accounts?.names || [];

  // Check if account ID is in the exclude list
  if (excludeIds.some((id) => id.toLowerCase() === account.id.toLowerCase())) {
    return true;
  }

  // Check if account name matches any exclude pattern (with wildcard support)
  for (const pattern of excludeNames) {
    const regex = wildcardToRegex(pattern);
    if (regex.test(account.name)) {
      return true;
    }
  }

  return false;
}

export function shouldExcludeRegion(region: string, config: AppConfig): boolean {
  const excludeRegions = config.exclude?.regions || [];
  return excludeRegions.some((r) => r.toLowerCase() === region.toLowerCase());
}
