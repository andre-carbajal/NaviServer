import { describe, expect, it, vi } from 'vitest';

import {
  getDroppedUploadEntries,
  getRootFolders,
  getUploadEntries,
} from './uploadEntries';

const fileWithPath = (name: string, relativePath?: string) => {
  const file = new File(['content'], name, { type: 'text/plain' });
  if (relativePath) {
    Object.defineProperty(file, 'webkitRelativePath', {
      configurable: true,
      value: relativePath,
    });
  }
  return file;
};

const fileEntry = (file: File) => ({
  isFile: true,
  isDirectory: false,
  name: file.name,
  file: (callback: (value: File) => void) => callback(file),
});

const directoryEntry = (
  name: string,
  batches: Array<Array<ReturnType<typeof fileEntry>>>,
) => ({
  isFile: false,
  isDirectory: true,
  name,
  createReader: () => ({
    readEntries: (
      callback: (entries: Array<ReturnType<typeof fileEntry>>) => void,
    ) => callback(batches.shift() || []),
  }),
});

describe('upload entry normalization', () => {
  it('normalizes input files and preserves folder roots', () => {
    const entries = getUploadEntries([
      fileWithPath('one.txt', 'folder/one.txt'),
      fileWithPath('two.txt', 'folder/nested/two.txt'),
    ]);

    expect(entries.map((entry) => entry.relativePath)).toEqual([
      'folder/one.txt',
      'folder/nested/two.txt',
    ]);
    expect([...getRootFolders(entries)]).toEqual(['folder']);
  });

  it('reads every directory batch when dropping a folder', async () => {
    const first = fileEntry(fileWithPath('one.txt'));
    const second = fileEntry(fileWithPath('two.txt'));
    const directory = directoryEntry('folder', [[first], [second], []]);
    const item = {
      kind: 'file',
      webkitGetAsEntry: vi.fn(() => directory),
      getAsFile: vi.fn(() => null),
    };

    const entries = await getDroppedUploadEntries([
      item,
    ] as unknown as DataTransferItemList);

    expect(entries.map((entry) => entry.relativePath)).toEqual([
      'folder/one.txt',
      'folder/two.txt',
    ]);
  });

  it('falls back to the dropped File when no entry is available', async () => {
    const file = fileWithPath('single.txt');
    const item = {
      kind: 'file',
      webkitGetAsEntry: () => null,
      getAsFile: () => file,
    };

    const entries = await getDroppedUploadEntries([
      item,
    ] as unknown as DataTransferItemList);

    expect(entries).toEqual([{ file }]);
  });
});
