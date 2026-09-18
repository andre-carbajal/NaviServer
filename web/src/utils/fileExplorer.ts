const ignoredEditExtensions = new Set([
  '.jar',
  '.zip',
  '.tar',
  '.gz',
  '.png',
  '.jpg',
  '.jpeg',
  '.gif',
  '.ico',
  '.exe',
  '.dll',
  '.so',
  '.dylib',
  '.ds_store',
]);

export const isEditableFile = (filename: string) => {
  const lower = filename.toLowerCase();
  if (lower === '.ds_store') return false;
  return !ignoredEditExtensions.has(lower.slice(lower.lastIndexOf('.')));
};

export const joinServerPath = (directory: string, name: string) =>
  directory === '/' ? `/${name}` : `${directory}/${name}`;

export const getParentServerPath = (path: string) =>
  path === '/' ? '/' : path.split('/').slice(0, -1).join('/') || '/';

export const formatFileSize = (bytes: number) => {
  if (bytes === 0) return '0 B';
  const unit = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const index = Math.floor(Math.log(bytes) / Math.log(unit));
  return `${Number.parseFloat((bytes / Math.pow(unit, index)).toFixed(2))} ${sizes[index]}`;
};
