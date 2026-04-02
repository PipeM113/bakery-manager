import { useState, useRef } from "react";
import type { DragEvent } from "react";
import { recipeService } from "../api/recipeService";

interface Props {
  recipeId: string;
  currentPhotoUrl: string | null;
  onUploaded: (photoUrl: string) => void;
}

const ALLOWED_TYPES = ["image/jpeg", "image/png", "image/webp"];
const MAX_SIZE = 5 * 1024 * 1024;

export default function RecipePhotoUpload({ recipeId, currentPhotoUrl, onUploaded }: Props) {
  const [preview, setPreview] = useState<string | null>(null);
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState("");
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const validate = (file: File): string => {
    if (!ALLOWED_TYPES.includes(file.type)) return "Solo se permiten archivos JPEG, PNG o WebP";
    if (file.size > MAX_SIZE) return "El archivo no puede superar los 5 MB";
    return "";
  };

  const handleFile = (file: File) => {
    const err = validate(file);
    if (err) {
      setError(err);
      setPreview(null);
      setPendingFile(null);
      return;
    }
    setError("");
    setPendingFile(file);
    const reader = new FileReader();
    reader.onload = (e) => setPreview(e.target?.result as string);
    reader.readAsDataURL(file);
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) handleFile(file);
  };

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files?.[0];
    if (file) handleFile(file);
  };

  const handleUpload = async () => {
    if (!pendingFile) return;
    setUploading(true);
    setError("");
    try {
      const { photo_url } = await recipeService.uploadPhoto(recipeId, pendingFile);
      setPreview(null);
      setPendingFile(null);
      if (fileInputRef.current) fileInputRef.current.value = "";
      onUploaded(photo_url);
    } catch {
      setError("Error al subir la foto. Intenta nuevamente.");
    } finally {
      setUploading(false);
    }
  };

  const handleCancel = () => {
    setPreview(null);
    setPendingFile(null);
    setError("");
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const displayPhoto = preview || currentPhotoUrl;

  return (
    <div className="space-y-3">
      <label className="block text-xs tracking-widest uppercase text-stone-400 font-light">
        Foto
      </label>

      {displayPhoto && (
        <div className="relative">
          <img
            src={displayPhoto}
            alt="Foto de receta"
            className="w-full h-44 object-cover border border-stone-200"
          />
          {preview && (
            <span className="absolute top-2 left-2 bg-gold text-white text-xs px-2 py-0.5 tracking-wide">
              Vista previa
            </span>
          )}
        </div>
      )}

      <div
        onDrop={handleDrop}
        onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
        onDragLeave={() => setDragOver(false)}
        onClick={() => !uploading && fileInputRef.current?.click()}
        className={`border-2 border-dashed cursor-pointer py-6 text-center transition-colors ${
          dragOver
            ? "border-gold bg-vanilla-100"
            : "border-stone-200 hover:border-stone-400"
        } ${uploading ? "opacity-50 cursor-not-allowed" : ""}`}
      >
        <p className="text-stone-400 text-xs font-light tracking-wide">
          {displayPhoto ? "Arrastra o haz clic para reemplazar" : "Arrastra o haz clic para subir foto"}
        </p>
        <p className="text-stone-300 text-xs mt-1">JPEG · PNG · WebP · máx 5 MB</p>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/jpeg,image/png,image/webp"
          onChange={handleInputChange}
          className="hidden"
          disabled={uploading}
        />
      </div>

      {error && (
        <p className="text-red-500 text-xs tracking-wide">{error}</p>
      )}

      {pendingFile && !uploading && (
        <div className="flex gap-2">
          <button
            onClick={handleUpload}
            className="flex-1 bg-gold text-white text-xs tracking-widest uppercase font-medium py-2.5 hover:bg-gold-light transition-all"
          >
            Subir foto
          </button>
          <button
            onClick={handleCancel}
            className="px-4 border border-stone-200 text-stone-500 text-xs tracking-widest uppercase font-medium py-2.5 hover:border-stone-400 transition-all"
          >
            Cancelar
          </button>
        </div>
      )}

      {uploading && (
        <div className="flex items-center justify-center gap-2 py-2.5 bg-vanilla-100">
          <svg className="animate-spin h-4 w-4 text-gold" viewBox="0 0 24 24" fill="none">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
          <span className="text-xs text-stone-500 font-light tracking-wide">Subiendo foto...</span>
        </div>
      )}
    </div>
  );
}
