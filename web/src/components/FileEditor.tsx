import Editor, { type OnMount } from '@monaco-editor/react';
import { ArrowLeft, FileCode, Loader2, Save } from 'lucide-react';
import { registerJson5Language } from 'monaco-json5-highlighter';

import React, { useEffect, useState } from 'react';

import { api } from '../services/api';

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
      alert('File saved successfully!');
    } catch (err) {
      const error = err as Error & { response?: { data?: string } };
      alert(error.response?.data || 'Failed to save file');
    } finally {
      setSaving(false);
    }
  };

  const hasChanges = content !== originalContent;
  const fileName = filePath.split('/').pop();

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

  const handleEditorMount: OnMount = (editor, monaco) => {
    registerJson5Language(monaco);
    editor.focus();
  };

  return (
    <div className="file-explorer-container">
      <div className="editor-header">
        <div className="editor-title">
          <button
            onClick={onClose}
            className="toolbar-btn"
            title="Back to files"
          >
            <ArrowLeft size={20} />
          </button>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <FileCode size={20} style={{ color: '#818cf8' }} />
            <span style={{ fontWeight: 500, color: 'white' }}>{fileName}</span>
            <span
              style={{
                fontSize: '0.75rem',
                color: '#6b7280',
                fontFamily: 'monospace',
              }}
              className="hidden sm:inline"
            >
              {filePath}
            </span>
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          {hasChanges && (
            <span className="unsaved-badge">
              <div className="unsaved-dot"></div>
              Unsaved Changes
            </span>
          )}
          <button
            onClick={handleSave}
            disabled={loading || saving || !hasChanges}
            className={`btn btn-primary ${!hasChanges || loading || saving ? 'disabled' : ''}`}
            style={{
              opacity: !hasChanges ? 0.5 : 1,
              cursor: !hasChanges ? 'not-allowed' : 'pointer',
            }}
          >
            {saving ? (
              <>
                <Loader2 className="spin" size={16} />
                Saving...
              </>
            ) : (
              <>
                <Save size={16} />
                Save
              </>
            )}
          </button>
        </div>
      </div>

      <div style={{ flex: 1, overflow: 'hidden', position: 'relative' }}>
        {loading ? (
          <div
            style={{
              display: 'flex',
              justifyContent: 'center',
              alignItems: 'center',
              height: '100%',
              color: '#6b7280',
            }}
          >
            <Loader2 className="spin" size={32} />
          </div>
        ) : error ? (
          <div
            style={{
              display: 'flex',
              justifyContent: 'center',
              alignItems: 'center',
              height: '100%',
              color: '#f87171',
              padding: '32px',
              textAlign: 'center',
              backgroundColor: 'rgba(127, 29, 29, 0.1)',
            }}
          >
            <div>
              <p style={{ marginBottom: '8px', fontWeight: 600 }}>
                Error Loading File
              </p>
              <p style={{ fontSize: '0.875rem', opacity: 0.8 }}>{error}</p>
              <button
                onClick={onClose}
                style={{
                  marginTop: '16px',
                  color: '#818cf8',
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  textDecoration: 'underline',
                }}
              >
                Go back
              </button>
            </div>
          </div>
        ) : (
          <Editor
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
        )}
      </div>

      <div className="editor-footer">
        <span>Space: 2</span>
        <span>UTF-8</span>
        <span>{getLanguage(filePath).toUpperCase()}</span>
      </div>
    </div>
  );
};

export default FileEditor;
