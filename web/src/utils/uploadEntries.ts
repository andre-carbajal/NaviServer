export interface UploadEntry {
  file: File;
  relativePath?: string;
}

interface FileSystemEntry {
  isFile: boolean;
  isDirectory: boolean;
  name: string;
  file: (callback: (file: File) => void) => void;
  createReader: () => {
    readEntries: (callback: (entries: FileSystemEntry[]) => void) => void;
  };
}

export const getUploadEntries = (files: FileList | File[]): UploadEntry[] =>
  Array.from(files).map((file) => ({
    file,
    relativePath: file.webkitRelativePath || undefined,
  }));

const readDirectoryEntries = async (
  reader: ReturnType<FileSystemEntry['createReader']>,
) => {
  const entries: FileSystemEntry[] = [];
  while (true) {
    const batch = await new Promise<FileSystemEntry[]>((resolve) => {
      reader.readEntries(resolve);
    });
    if (batch.length === 0) return entries;
    entries.push(...batch);
  }
};

const traverseEntry = async (
  entry: FileSystemEntry,
  parentPath = '',
): Promise<UploadEntry[]> => {
  const entryPath = parentPath ? `${parentPath}/${entry.name}` : entry.name;
  if (entry.isFile) {
    const file = await new Promise<File>((resolve) => entry.file(resolve));
    return [{ file, relativePath: parentPath ? entryPath : undefined }];
  }
  if (!entry.isDirectory) return [];

  const children = await readDirectoryEntries(entry.createReader());
  const nested = await Promise.all(
    children.map((child) => traverseEntry(child, entryPath)),
  );
  return nested.flat();
};

export const getDroppedUploadEntries = async (
  items: DataTransferItemList,
): Promise<UploadEntry[]> => {
  const entries: UploadEntry[] = [];
  for (const item of Array.from(items)) {
    if (item.kind !== 'file') continue;
    const entry = item.webkitGetAsEntry() as FileSystemEntry | null;
    if (entry) {
      entries.push(...(await traverseEntry(entry)));
      continue;
    }
    const file = item.getAsFile();
    if (file) entries.push({ file });
  }
  return entries;
};

export const getRootFolders = (entries: UploadEntry[]) =>
  new Set(
    entries
      .map(({ relativePath }) => relativePath?.split('/')[0])
      .filter((name): name is string => Boolean(name)),
  );
