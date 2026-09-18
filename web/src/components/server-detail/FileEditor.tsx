import type { OnMount } from '@monaco-editor/react';
import { ArrowLeft, FileCode, Loader2, Save } from 'lucide-react';
import { registerJson5Language } from 'monaco-json5-highlighter';

import React, { Suspense, useEffect, useState } from 'react';

import { useModalDialog } from '../../hooks/useModalDialog';
import { api } from '../../services/api';
import { Button } from '../ui/Button';

const getLanguage = (path: string) => {
  const ext = path.split('.').pop()?.toLowerCase();
  switch (ext) {
    case 'json':
      return 'json';
    case 'json5':
      return 'json5';
    case 'js':
    case 'jsx':
      return 'javascript';
    case 'ts':
    case 'tsx':
      return 'typescript';
    case 'html':
      return 'html';
    case 'css':
      return 'css';
    case 'xml':
      return 'xml';
    case 'yaml':
    case 'yml':
      return 'yaml';
    case 'properties':
    case 'ini':
    case 'conf':
    case 'toml':
      return 'ini';
    case 'sh':
    case 'bash':
      return 'shell';
    default:
      return 'plaintext';
  }
};

const MonacoEditor = React.lazy(async () => {
  const module = await import('@monaco-editor/react');
  return { default: module.default };
});

const handleEditorMount: OnMount = (editor, monaco) => {
  registerJson5Language(monaco);
  editor.focus();
};

interface FileEditorProps {
  serverId: string;
  filePath: string;
  onClose: () => void;
}

const FileEditor: React.FC<FileEditorProps> = ({
  serverId,
  filePath,
  onClose,
}) => {
  const [content, setContent] = useState('');
  const [originalContent, setOriginalContent] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { showAlert, modalDialog } = useModalDialog();

  useEffect(() => {
    const loadContent = async () => {
      setLoading(true);
      setError(null);
      try {
        const res = await api.getFileContent(serverId, filePath);
        const text =
          typeof res.data === 'string'
            ? res.data
            : JSON.stringify(res.data, null, 2);
        setContent(text);
        setOriginalContent(text);
      } catch (err) {
        const error = err as Error & { response?: { data?: string } };
        setError(
          error.response?.data ||
            error.message ||
            'Failed to load file content',
        );
      } finally {
        setLoading(false);
      }
    };

    if (filePath) {
      loadContent();
    }
  }, [serverId, filePath]);

  const handleSave = async () => {
    setSaving(true);
    try {
      await api.saveFileContent(serverId, filePath, content);
      setOriginalContent(content);
      await showAlert({
        title: 'File Saved',
        message: 'File saved successfully.',
        variant: 'success',
      });
    } catch (err) {
      const error = err as Error & { response?: { data?: string } };
      await showAlert({
        title: 'Save Failed',
        message: error.response?.data || 'Failed to save file.',
        variant: 'danger',
      });
    } finally {
      setSaving(false);
    }
  };

  const hasChanges = content !== originalContent;
  const fileName = filePath.split('/').pop();

  const renderEditorContent = () => {
    if (loading) {
      return (
        <div className="tw:flex tw:h-full tw:items-center tw:justify-center tw:text-gray-500">
          <Loader2 className="tw:animate-spin" size={32} />
        </div>
      );
    }

    if (error) {
      return (
        <div className="tw:flex tw:h-full tw:items-center tw:justify-center tw:bg-red-900/10 tw:p-8 tw:text-center tw:text-red-400">
          <div>
            <p className="tw:mb-2 tw:font-semibold">Error Loading File</p>
            <p className="tw:text-sm tw:opacity-80">{error}</p>
            <button
              type="button"
              onClick={onClose}
              className="tw:mt-4 tw:cursor-pointer tw:border-0 tw:bg-transparent tw:text-indigo-400 tw:underline"
            >
              Go back
            </button>
          </div>
        </div>
      );
    }

    return (
      <Suspense
        fallback={
          <div className="tw:flex tw:h-full tw:items-center tw:justify-center">
            <Loader2 className="tw:animate-spin" size={32} />
          </div>
        }
      >
        <MonacoEditor
          height="100%"
          defaultLanguage={getLanguage(filePath)}
          language={getLanguage(filePath)}
          value={content}
          theme="vs-dark"
          onChange={(value) => setContent(value || '')}
          onMount={handleEditorMount}
          options={{
            minimap: { enabled: true },
            fontSize: 14,
            scrollBeyondLastLine: false,
            automaticLayout: true,
            tabSize: 2,
            wordWrap: 'on',
          }}
        />
      </Suspense>
    );
  };

  return (
    <div className="tw:flex tw:h-[600px] tw:flex-col tw:rounded-lg tw:border tw:border-border tw:bg-[#1e1e1e] tw:shadow-sm tw:max-[769px]:h-[400px] tw:max-[481px]:h-[300px]">
      {modalDialog}
      <div className="tw:flex tw:items-center tw:justify-between tw:border-b tw:border-border tw:p-4">
        <div className="tw:flex tw:items-center tw:gap-2">
          <button
            type="button"
            onClick={onClose}
            className="tw:flex tw:cursor-pointer tw:items-center tw:justify-center tw:rounded-md tw:border-0 tw:bg-transparent tw:p-2 tw:text-gray-400 tw:transition-all tw:duration-200 tw:hover:bg-[#2d2d2d] tw:hover:text-white tw:disabled:cursor-not-allowed tw:disabled:bg-transparent tw:disabled:opacity-50 tw:disabled:text-gray-400"
            title="Back to files"
          >
            <ArrowLeft size={20} />
          </button>
          <div className="tw:flex tw:items-center tw:gap-2">
            <FileCode size={20} className="tw:text-indigo-400" />
            <span className="tw:font-medium tw:text-white">{fileName}</span>
            <span className="tw:hidden tw:font-mono tw:text-xs tw:text-gray-500 tw:sm:inline">
              {filePath}
            </span>
          </div>
        </div>
        <div className="tw:flex tw:items-center tw:gap-4">
          {hasChanges && (
            <span className="tw:flex tw:items-center tw:gap-1.5 tw:text-xs tw:text-amber-300">
              <div className="tw:h-2 tw:w-2 tw:rounded-full tw:bg-amber-300"></div>
              <span className="tw:hidden tw:sm:inline">Unsaved Changes</span>
            </span>
          )}
          <Button
            onClick={handleSave}
            disabled={loading || saving || !hasChanges}
          >
            {saving ? (
              <>
                <Loader2 className="tw:animate-spin" size={16} />
                <span className="tw:hidden tw:sm:inline">Saving...</span>
              </>
            ) : (
              <>
                <Save size={16} />
                <span className="tw:hidden tw:sm:inline">Save</span>
              </>
            )}
          </Button>
        </div>
      </div>

      <div className="tw:relative tw:flex-1 tw:overflow-hidden">
        {renderEditorContent()}
      </div>

      <div className="tw:flex tw:justify-between tw:border-t tw:border-border tw:bg-[#252525] tw:p-2 tw:text-xs tw:text-gray-500">
        <span>Space: 2</span>
        <span>UTF-8</span>
        <span>{getLanguage(filePath).toUpperCase()}</span>
      </div>
    </div>
  );
};

export default FileEditor;
