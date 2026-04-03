import React, { useEffect, useRef, useState } from 'react';

import { api } from '../services/api';
import type { Server } from '../types';
import { Button } from './ui/Button';
import { Modal } from './ui/Modal';

interface EditServerModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (data: {
    name: string;
    ram: number;
    customArgs?: string;
    icon?: File;
  }) => Promise<void>;
  server: Server | null;
}

const EditServerModal: React.FC<EditServerModalProps> = ({
  isOpen,
  onClose,
  onSave,
  server,
}) => {
  const [name, setName] = useState('');
  const [ram, setRam] = useState(0);
  const [customArgs, setCustomArgs] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [selectedIcon, setSelectedIcon] = useState<File | null>(null);
  const [iconPreview, setIconPreview] = useState<string | null>(null);
  const [imageError, setImageError] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isOpen && server) {
      setName((prev) => (prev === '' ? server.name : prev));
      setRam((prev) => (prev === 0 ? server.ram : prev));
      setCustomArgs((prev) => (prev === '' ? server.customArgs || '' : prev));
    }

    if (!isOpen) {
      setName('');
      setRam(0);
      setCustomArgs('');
      setSelectedIcon(null);
      setIconPreview(null);
      setImageError(false);
    }
  }, [isOpen, server]);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      const file = e.target.files[0];
      setSelectedIcon(file);
      const reader = new FileReader();
      reader.onloadend = () => {
        setIconPreview(reader.result as string);
      };
      reader.readAsDataURL(file);
    }
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsSaving(true);
    try {
      await onSave({ name, ram, customArgs, icon: selectedIcon || undefined });
      onClose();
    } catch (error) {
      console.error('Failed to save server settings:', error);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Edit Server">
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label>Server Icon</label>
          <div style={{ display: 'flex', alignItems: 'center', gap: '15px' }}>
            <div
              style={{
                width: '64px',
                height: '64px',
                borderRadius: '8px',
                background: '#333',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                overflow: 'hidden',
                border: '1px solid #444',
                position: 'relative',
              }}
            >
              {iconPreview ? (
                <img
                  src={iconPreview}
                  alt="Preview"
                  style={{
                    width: '100%',
                    height: '100%',
                    objectFit: 'fill',
                    imageRendering: 'pixelated',
                  }}
                />
              ) : !imageError && server ? (
                <img
                  src={`${api.getServerIconUrl(server.id)}?t=${Date.now()}`}
                  alt="Current"
                  onError={() => setImageError(true)}
                  style={{
                    width: '100%',
                    height: '100%',
                    objectFit: 'contain',
                    imageRendering: 'pixelated',
                  }}
                />
              ) : (
                <span
                  style={{
                    fontSize: '24px',
                    color: '#666',
                  }}
                >
                  {server?.name.charAt(0).toUpperCase()}
                </span>
              )}
            </div>
            <div style={{ flex: 1 }}>
              <input
                type="file"
                accept="image/png,image/jpeg"
                onChange={handleFileChange}
                ref={fileInputRef}
                style={{ display: 'none' }}
              />
              <Button
                type="button"
                variant="secondary"
                onClick={() => fileInputRef.current?.click()}
              >
                Choose Image
              </Button>
              <div
                style={{
                  fontSize: '0.8rem',
                  color: '#666',
                  marginTop: '5px',
                }}
              >
                Recommended 64x64 PNG
              </div>
            </div>
          </div>
        </div>

        <div className="form-group">
          <label>Server Name</label>
          <input
            type="text"
            className="form-input"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
        </div>
        <div className="form-group">
          <label>RAM (MB)</label>
          <input
            type="number"
            className="form-input"
            value={ram}
            onChange={(e) => setRam(parseInt(e.target.value, 10))}
            required
            min="1024"
            step="512"
          />
        </div>
        <div className="form-group">
          <label>Custom Arguments</label>
          <input
            type="text"
            className="form-input"
            value={customArgs}
            onChange={(e) => setCustomArgs(e.target.value)}
            placeholder="-Dexample=true"
          />
        </div>
        <div className="modal-actions">
          <Button
            type="button"
            variant="secondary"
            onClick={onClose}
            disabled={isSaving}
          >
            Cancel
          </Button>
          <Button type="submit" disabled={isSaving}>
            {isSaving ? 'Saving...' : 'Save'}
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default EditServerModal;
