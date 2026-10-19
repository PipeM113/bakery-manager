import { useRef, useState } from "react";
import * as XLSX from "xlsx";
import api from "../services/api";

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

const EXPECTED_HEADERS = ["Nombre", "Marca", "Unidad", "Tamaño Presentación", "Precio Presentación"];

export default function IngredientImportModal({ onClose, onSuccess }: Props) {
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string[][]>([]);
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

    const reader = new FileReader();
    reader.onload = (evt) => {
      const data = evt.target?.result;
      const wb = XLSX.read(data, { type: "array" });
      const ws = wb.Sheets[wb.SheetNames[0]];
      const rows = XLSX.utils.sheet_to_json<string[]>(ws, { header: 1, defval: "" }) as string[][];
      setPreview(rows.slice(0, 6)); // header + up to 5 data rows
    };
    reader.readAsArrayBuffer(f);
  };

  const handleImport = async () => {
    if (!file) return;
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
    } catch (e: any) {
      setError(e.response?.data?.error || "Error al importar");
    } finally {
      setImporting(false);
    }
  };

  const succeeded = result && result.errors.length === 0;

  return (
    <div className="fixed inset-0 z-50 bg-negro bg-opacity-40 flex items-end md:items-center justify-center p-4">
      <div className="bg-white border border-linea w-full max-w-lg p-6 space-y-4 shadow-lg">

        {/* Header */}
        <div className="flex items-center justify-between">
          <h2 className="font-display text-lg text-negro">Importar desde Excel</h2>
          <button onClick={onClose} className="text-negro/70 hover:text-negro text-lg leading-none">×</button>
        </div>

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

        {/* Preview table */}
        {preview.length > 0 && !result && (
          <div className="overflow-x-auto border border-linea">
            <table className="w-full text-xs text-negro/70 font-light">
              <thead>
                <tr className="bg-lino border-b border-linea">
                  {(preview[0] || EXPECTED_HEADERS).map((h, i) => (
                    <th key={i} className="text-left py-1.5 px-2 text-negro/70 uppercase tracking-wider font-light whitespace-nowrap">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {preview.slice(1).map((row, i) => (
                  <tr key={i} className="border-b border-linea last:border-0">
                    {(preview[0] || EXPECTED_HEADERS).map((_, j) => (
                      <td key={j} className="py-1.5 px-2 whitespace-nowrap">{row[j] ?? ""}</td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
            {preview.length > 6 && (
              <p className="text-xs text-negro/70 font-light px-2 py-1">Mostrando primeras 5 filas de datos...</p>
            )}
          </div>
        )}

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

        {error && <p className="text-peligro text-xs tracking-wide">{error}</p>}

        {/* Actions */}
        <div className="flex gap-3 pt-1">
          {!succeeded && (
            <button
              onClick={handleImport}
              disabled={!file || importing}
              className="flex-1 bg-negro text-crema text-xs tracking-widest uppercase font-medium
                         py-2.5 hover:bg-black transition-all disabled:opacity-50"
            >
              {importing ? "Importando..." : "Importar"}
            </button>
          )}
          <button
            onClick={onClose}
            className="flex-1 border border-linea text-negro/70 text-xs tracking-widest uppercase
                       font-medium py-2.5 hover:border-oro hover:text-oro transition-all"
          >
            {succeeded ? "Cerrar" : "Cancelar"}
          </button>
        </div>
      </div>
    </div>
  );
}
