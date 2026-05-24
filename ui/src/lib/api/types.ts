export type StatusState = 'up' | 'degraded' | 'down' | 'unknown';
export type Source = 'httproute' | 'bookmark';

export interface Status {
  state: StatusState;
  statusCode: number;
  latencyMs: number;
  checkedAt: string;
  error?: string;
}

export interface GatewayRef {
  namespace: string;
  name: string;
}

export interface K8sInfo {
  namespace: string;
  httpRouteName: string;
  gatewayRefs: GatewayRef[];
}

export interface Tile {
  id: string;
  source: Source;
  name: string;
  url: string;
  icon: string;
  description?: string;
  group: string;
  order: number;
  hidden: boolean;
  insecureSkipVerify?: boolean;
  status: Status;
  k8s?: K8sInfo | null;
}

export interface Group {
  id: string;
  name: string;
  order: number;
}

export interface View {
  groups: Group[];
  tiles: Tile[];
}

export interface ConfigPatch {
  settings?: Settings;
  groups?: GroupSpec[];
  tiles?: TileOverride[];
  bookmarks?: Bookmark[];
}

export interface Settings {
  title?: string;
  theme?: 'dark' | 'light' | 'auto';
  healthCheck?: {
    enabled?: boolean;
    intervalSeconds?: number;
    timeoutSeconds?: number;
    insecureSkipVerify?: boolean;
  };
}

export interface GroupSpec {
  id: string;
  name: string;
  order: number;
}

export interface TileOverride {
  id: string;
  hidden?: boolean;
  name?: string;
  description?: string;
  icon?: string;
  group?: string;
  order?: number;
  url?: string;
  insecureSkipVerify?: boolean;
}

export interface Bookmark {
  id: string;
  name: string;
  url: string;
  icon?: string;
  group?: string;
  order?: number;
}
