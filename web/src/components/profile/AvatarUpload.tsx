/**
 * AvatarUpload — the clickable profile photo in the hero card.
 *
 * Click the avatar (or the 📷 overlay) to pick a file. The file is
 * resized client-side to a 256x256 JPEG before being sent to
 * `/api/v1/auth/profile` so we never blow past the 500KB backend
 * cap on full-resolution selfies.
 *
 * Owns its own upload state — parent doesn't need to know that
 * "we just resized the image, here's the data URL".
 */
import { FormEvent, useState } from 'react';
import { ApiError, api } from '../../lib/api';

interface AvatarUploadProps {
  currentUrl: string;
  email: string;
  userId: string;
  initials: string;
  avatarColor: (seed: string) => string;
  onUploaded: (dataUrl: string) => void;
  onRemoved: () => void;
}

async function fileToResizedDataUrl(file: File, maxSide = 256, quality = 0.7): Promise<string> {
  // Read a File from the browser File API, render it into a
  // hidden canvas, resize it to fit within maxSide x maxSide
  // preserving aspect ratio, and re-export as JPEG.
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => {
      try {
        const ratio = Math.min(1, maxSide / Math.max(img.width, img.height));
        const w = Math.max(1, Math.round(img.width * ratio));
        const h = Math.max(1, Math.round(img.height * ratio));
        const canvas = document.createElement('canvas');
        canvas.width = w;
        canvas.height = h;
        const ctx = canvas.getContext('2d');
        if (!ctx) throw new Error('canvas 2d unavailable');
        ctx.drawImage(img, 0, 0, w, h);
        const dataUrl = canvas.toDataURL('image/jpeg', quality);
        URL.revokeObjectURL(url);
        resolve(dataUrl);
      } catch (err) {
        URL.revokeObjectURL(url);
        reject(err);
      }
    };
    img.onerror = (err) => {
      URL.revokeObjectURL(url);
      reject(err);
    };
    img.src = url;
  });
}

export default function AvatarUpload({
  currentUrl,
  email,
  userId,
  initials,
  avatarColor,
  onUploaded,
  onRemoved,
}: AvatarUploadProps) {
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onPick(event: FormEvent<HTMLInputElement>) {
    const file = event.currentTarget.files?.[0];
    // Reset so picking the same file twice fires onChange again.
    event.currentTarget.value = '';
    if (!file) return;
    if (!file.type.startsWith('image/')) {
      setError('Please choose an image file (PNG, JPG, GIF, WebP).');
      return;
    }
    setError(null);
    setUploading(true);
    try {
      const dataUrl = await fileToResizedDataUrl(file, 256, 0.7);
      await api('PATCH', '/api/v1/auth/profile', { avatar_url: dataUrl });
      onUploaded(dataUrl);
    } catch (cause) {
      if (cause instanceof ApiError) {
        setError(cause.friendlyMessage);
      } else {
        setError(cause instanceof Error ? cause.message : 'Could not upload your photo.');
      }
    } finally {
      setUploading(false);
    }
  }

  async function onRemove() {
    if (!currentUrl) return;
    setUploading(true);
    setError(null);
    try {
      await api('PATCH', '/api/v1/auth/profile', { avatar_url: '' });
      onRemoved();
    } catch (cause) {
      if (cause instanceof ApiError) {
        setError(cause.friendlyMessage);
      } else {
        setError(cause instanceof Error ? cause.message : 'Could not remove your photo.');
      }
    } finally {
      setUploading(false);
    }
  }

  return (
    <div className="prof-hero-avatar-wrap">
      <label
        className={`prof-hero-avatar ${currentUrl ? 'has-image' : ''}`}
        style={currentUrl ? undefined : { background: avatarColor(email || userId) }}
        aria-hidden="true"
      >
        {currentUrl ? <img src={currentUrl} alt="" /> : <span>{initials}</span>}
        <span className="prof-hero-avatar-overlay" aria-hidden="true">📷</span>
        {uploading && <span className="prof-hero-avatar-spinner" aria-hidden="true" />}
      </label>
      <input
        type="file"
        accept="image/*"
        className="prof-hero-avatar-input"
        onChange={(e) => void onPick(e)}
        disabled={uploading}
        aria-label="Upload profile photo"
      />
      {currentUrl && !uploading && (
        <button
          type="button"
          className="prof-hero-avatar-remove"
          onClick={() => void onRemove()}
          aria-label="Remove profile photo"
          title="Remove photo"
        >×</button>
      )}
      {error && <div className="prof-hero-avatar-error">{error}</div>}
    </div>
  );
}
