import Modal from "./Modal";
import { useRef, useState } from "react";
import api from "../services/api";
import { apiErrorMessage } from "../utils/apiError";

interface ImportRowError {
  row: number;
  reason: string;
}

interface ImportResult {
  imported: number;
  updated: number;
  skipped: number;
  errors: ImportRowError[];
}

interface Props {
  onClose: () => void;
  onSuccess: () => void;
}

// The server accepts up to 4 MiB (docs/acceso.md).
const MAX_BYTES = 4 * 1024 * 1024;

const EXPECTED_HEADERS = ["Nombre", "Marca", "Unidad", "Tamaño Presentación", "Precio Presentación"];

export default function IngredientImportModal({ onClose, onSuccess }: Props) {
  const [file, setFile] = useState<File | null>(null);
  const [importing, setImporting] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [error, setError] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (!f) return;
    setFile(f);
    setResult(null);
    setError("");

    // The file is not read here: the server reads it and says what is wrong with it.
    if (f.size > MAX_BYTES) setError("El archivo pesa más de 4 MB");
  };

  const handleImport = async () => {
    if (!file || file.size > MAX_BYTES) return;
    setImporting(true);
    setError("");
    try {
      const formData = new FormData();
      formData.append("file", file);
      const res = await api.post("/ingredients/import", formData, {
        headers: { "Content-Type": "multipart/form-data" },
      });
      setResult(res.data);
      if (res.data.errors?.length === 0 && (res.data.imported > 0 || res.data.updated > 0)) {
        onSuccess();
      }
    } catch (e) {
      setError(apiErrorMessage(e, "Error al importar"));
    } finally {
      setImporting(false);
    }
  };

  const succeeded = result && result.errors.length === 0;

  return (
    <Modal
      title="Importar desde Excel"
      onClose={onClose}
      maxWidth="max-w-lg"
      footer={
        <div>
          {error && <p className="text-peligro text-xs tracking-wide mb-2">{error}</p>}
          <div className="flex gap-3 pt-1">
          {!succeeded && (
            <button
              onClick={handleImport}
              disabled={!file || importing}
              data-primary-action
              className="flex-1 bg-negro text-crema text-xs tracking-widest uppercase font-medium
                         py-2.5 hover:bg-black transition-all disabled:opacity-50"
            >
              {importing ? "Importando..." : "Importar"}
            </button>
          )}
          <button
            onClick={onClose}
            {...(succeeded ? { "data-primary-action": true } : {})}
            className="flex-1 border border-linea text-negro/70 text-xs tracking-widest uppercase
                       font-medium py-2.5 hover:border-oro hover:text-oro transition-all"
          >
            {succeeded ? "Cerrar" : "Cancelar"}
          </button>
        </div>
        </div>
      }
    >
        {/* Instructions */}
        <p className="text-xs text-negro/70 font-light leading-relaxed">
          El archivo .xlsx debe tener la fila 1 como encabezado con columnas:{" "}
          <span className="text-negro/70">{EXPECTED_HEADERS.join(", ")}</span>
        </p>

        {/* File input */}
        <div>
          <input ref={inputRef} type="file" accept=".xlsx" onChange={handleFileChange} className="hidden" />
          <button
            onClick={() => inputRef.current?.click()}
            className="w-full border border-dashed border-linea text-negro/70 text-xs tracking-widest uppercase
                       py-4 hover:border-oro hover:text-oro transition-all font-light"
          >
            {file ? file.name : "Seleccionar archivo .xlsx"}
          </button>
        </div>

        {/* Result */}
        {result && (
          <div className="space-y-2">
            {succeeded ? (
              <div className="bg-salvia/40 border border-salvia p-3 text-xs text-negro space-y-1">
                {result.imported > 0 && (
                  <p className="font-medium">{result.imported} ingrediente{result.imported !== 1 ? "s" : ""} nuevo{result.imported !== 1 ? "s" : ""} agregado{result.imported !== 1 ? "s" : ""}</p>
                )}
                {result.updated > 0 && (
                  <p className="font-medium">{result.updated} ingrediente{result.updated !== 1 ? "s" : ""} actualizado{result.updated !== 1 ? "s" : ""}</p>
                )}
                {result.imported === 0 && result.updated === 0 && (
                  <p className="font-medium">Sin cambios</p>
                )}
                {result.skipped > 0 && (
                  <p className="text-negro font-light">{result.skipped} fila{result.skipped !== 1 ? "s" : ""} vacía{result.skipped !== 1 ? "s" : ""} omitida{result.skipped !== 1 ? "s" : ""}</p>
                )}
              </div>
            ) : (
              <div className="space-y-2">
                <div className="bg-rosa/30 border border-peligro p-3 text-xs text-peligro">
                  <p className="font-medium">No se importó nada. {result.errors.length} error{result.errors.length !== 1 ? "es" : ""} encontrado{result.errors.length !== 1 ? "s" : ""}:</p>
                </div>
                <div className="max-h-40 overflow-y-auto space-y-1 border border-linea p-2">
                  {result.errors.map((e, i) => (
                    <div key={i} className="text-xs text-negro/70 font-light flex gap-2">
                      <span className="text-negro/70 shrink-0 font-medium">Fila {e.row}:</span>
                      <span>{e.reason}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}


    </Modal>
  );
}
