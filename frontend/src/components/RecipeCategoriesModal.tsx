import { useEffect, useRef, useState } from "react";
import Modal from "./Modal";
import { recipeCategoryService } from "../api/recipeCategoryService";
import type { IRecipeType } from "../types/recipe";
import { apiErrorMessage } from "../utils/apiError";
import { sortKey } from "../utils/ingredients";

interface Props {
  onClose: () => void;
  /** Called after any change, so the screen behind reloads its recipes and types. */
  onChanged: () => void;
}

const field = "w-full bg-crema border border-linea text-negro px-3 py-2 text-base font-light focus:outline-none focus:border-oro transition-all";
const link = "text-xs text-negro/70 hover:text-oro tracking-wider transition-colors";

/** The window that manages the types of recipe: create, rename and delete (the recipes of a deleted type stay, without type). */
export default function RecipeCategoriesModal({ onClose, onChanged }: Props) {
  const [types, setTypes]       = useState<IRecipeType[]>([]);
  const [loaded, setLoaded]     = useState(false);
  const [newName, setNewName]   = useState("");
  const [renaming, setRenaming] = useState<string | null>(null);
  const [draft, setDraft]       = useState("");
  const [deleting, setDeleting] = useState<string | null>(null);
  const [error, setError]       = useState("");
  const [busy, setBusy]         = useState(false);
  const working = useRef(false); // set at once, before the screen re-renders: a double Enter sends one request

  const reload = () =>
    recipeCategoryService.getAll()
      .then((all) => setTypes([...all].sort((a, b) => sortKey(a.name).localeCompare(sortKey(b.name)))))
      .catch((e) => setError(apiErrorMessage(e, "No se pudieron cargar los tipos")))
      .finally(() => setLoaded(true));

  useEffect(() => { reload(); }, []);

  const run = async (action: () => Promise<unknown>, fallback: string) => {
    if (working.current) return false;
    working.current = true;
    setError("");
    setBusy(true);
    try {
      await action();
      await reload();
      onChanged();
      return true;
    } catch (e) {
      setError(apiErrorMessage(e, fallback));
      return false;
    } finally {
      working.current = false;
      setBusy(false);
    }
  };

  const create = async () => {
    if (!newName.trim()) { setError("Escribe el nombre del tipo"); return; }
    if (await run(() => recipeCategoryService.create(newName), "No se pudo crear el tipo")) setNewName("");
  };

  const rename = async (id: string) => {
    if (!draft.trim()) { setError("Escribe el nombre del tipo"); return; }
    if (await run(() => recipeCategoryService.rename(id, draft), "No se pudo renombrar el tipo")) setRenaming(null);
  };

  const remove = async (id: string) => {
    if (await run(() => recipeCategoryService.delete(id), "No se pudo eliminar el tipo")) setDeleting(null);
  };

  const consequence = (t: IRecipeType) =>
    t.recipe_count === 0
      ? "No tiene recetas."
      : `${t.recipe_count} ${t.recipe_count === 1 ? "receta quedará" : "recetas quedarán"} sin tipo. Las recetas no se borran.`;

  return (
    <Modal
      title="Tipos de receta"
      onClose={onClose}
      footer={
        <button onClick={onClose} data-primary-action
          className="w-full bg-negro text-crema text-xs tracking-widest uppercase font-medium py-2.5 hover:bg-black transition-all">
          Listo
        </button>
      }
    >
      {error && <p role="alert" className="bg-rosa text-negro text-sm font-light px-3 py-2">{error}</p>}

      <p className="text-xs text-negro/70 font-light">
        Sirven para agrupar las recetas (por ejemplo, bizcocho, hojarasca o merengue). Son opcionales.
      </p>

      {loaded && types.length === 0 && <p className="text-sm text-negro/70 font-light">Todavía no hay tipos.</p>}

      <ul className="space-y-2">
        {types.map((t) => (
          <li key={t.id} data-testid="type-row" className="border border-linea p-3 space-y-2">
            {renaming === t.id ? (
              <div className="space-y-2">
                <input aria-label="Nuevo nombre" className={field} value={draft} maxLength={50}
                  onChange={(e) => setDraft(e.target.value)} />
                <div className="flex gap-3">
                  <button disabled={busy} onClick={() => rename(t.id)} className={`${link} text-negro`}>Guardar</button>
                  <button onClick={() => { setRenaming(null); setError(""); }} className={link}>Cancelar</button>
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-between gap-3">
                <p className="text-sm text-negro break-words min-w-0">
                  {t.name} <span className="text-xs text-negro/70 font-light">{t.recipe_count}</span>
                </p>
                <div className="flex gap-3 shrink-0">
                  <button onClick={() => { setRenaming(t.id); setDraft(t.name); setDeleting(null); setError(""); }} className={link}>Renombrar</button>
                  <button onClick={() => { setDeleting(t.id); setRenaming(null); setError(""); }} className={`${link} hover:text-peligro`}>Eliminar</button>
                </div>
              </div>
            )}
            {deleting === t.id && (
              <div className="bg-lino px-3 py-2 space-y-2">
                <p className="text-sm text-negro font-light">{consequence(t)}</p>
                <div className="flex gap-3">
                  <button disabled={busy} onClick={() => remove(t.id)} className={`${link} text-peligro`}>Sí, eliminar</button>
                  <button onClick={() => setDeleting(null)} className={link}>Cancelar</button>
                </div>
              </div>
            )}
          </li>
        ))}
      </ul>

      {renaming === null && (
        <div className="space-y-2 pt-1">
          <label className="block text-xs tracking-widest uppercase text-negro/70 font-light">Nuevo tipo</label>
          <div className="flex gap-2">
            <input aria-label="Nombre del tipo" placeholder="Nombre del tipo" className={field} value={newName} maxLength={50}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") create(); }} />
            <button disabled={busy} onClick={create}
              className="border border-linea text-negro text-xs tracking-widest uppercase font-medium px-4 hover:border-oro hover:text-oro transition-all shrink-0">
              Crear
            </button>
          </div>
        </div>
      )}
    </Modal>
  );
}
