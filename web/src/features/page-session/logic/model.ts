import type { OutlineDocument } from '../../../entities/outline';
import type { RemotePage } from './ports';

export type SaveState = 'saved' | 'saving' | 'error' | 'conflict' | 'revoked';

export interface SessionIdentity {
  pageId: string;
  urlSecret: string;
  locator: string;
}

export interface OpenSession extends SessionIdentity {
  document: OutlineDocument;
  password: string;
  writeToken: string;
  revision: number;
  remotePage: RemotePage;
  epoch: number;
}

export type AppViewState =
  | { kind: 'start' }
  | { kind: 'loading'; identity: SessionIdentity }
  | { kind: 'locked'; identity: SessionIdentity; remotePage: RemotePage }
  | { kind: 'missing'; identity: SessionIdentity }
  | { kind: 'open'; session: OpenSession; saveState: SaveState };
