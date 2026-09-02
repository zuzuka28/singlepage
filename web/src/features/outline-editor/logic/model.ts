import type { OutlineDocument } from '../../../entities/outline';

export type FocusCaret = 'start' | 'end' | number;
export type FocusIntent = { id: string; caret?: FocusCaret } | null;

export interface EditorCommandResult {
  document: OutlineDocument;
  focusIntent: FocusIntent;
}
