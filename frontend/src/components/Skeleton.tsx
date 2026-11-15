// Placeholders shown while a screen loads: they keep the shape of the page (header, cards, rows),
// so nothing jumps when the data arrives, and they say "Cargando" to screen readers.
// The pulse stops for people who asked their system for less motion.
const block = "bg-lino motion-safe:animate-pulse";

export function SkeletonBlock({ className = "" }: { className?: string }) {
  return <div aria-hidden="true" className={`${block} ${className}`} />;
}

function Frame({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div role="status" aria-busy="true" aria-live="polite" data-testid="skeleton" className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">
      <span className="sr-only">{label}</span>
      {children}
    </div>
  );
}

function Header() {
  return (
    <div className="mb-8">
      <SkeletonBlock className="h-3 w-28 mb-4" />
      <SkeletonBlock className="h-9 w-64 max-w-full" />
    </div>
  );
}

/** A list of rows (ingredients, recipes, recent recipes). */
export function ListSkeleton({ rows = 6, label = "Cargando…" }: { rows?: number; label?: string }) {
  return (
    <Frame label={label}>
      <Header />
      <SkeletonBlock className="h-12 w-full mb-6" />
      <div className="space-y-2">
        {Array.from({ length: rows }, (_, i) => (
          <div key={i} className="bg-white border border-linea p-4 flex items-center gap-4">
            <SkeletonBlock className="h-14 w-14 shrink-0" />
            <div className="flex-1 space-y-2">
              <SkeletonBlock className="h-4 w-2/3" />
              <SkeletonBlock className="h-3 w-1/3" />
            </div>
          </div>
        ))}
      </div>
    </Frame>
  );
}

/** Figures plus a list (dashboard, profitability). */
export function CardsSkeleton({ cards = 4, rows = 4, label = "Cargando…", bare = false }: { cards?: number; rows?: number; label?: string; bare?: boolean }) {
  const body = (
    <>
      <div className="grid grid-cols-1 gap-4 mb-10 min-[480px]:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: cards }, (_, i) => (
          <div key={i} className="bg-white border border-linea p-5 space-y-3">
            <SkeletonBlock className="h-7 w-24" />
            <SkeletonBlock className="h-3 w-32" />
          </div>
        ))}
      </div>
      <div className="space-y-2">
        {Array.from({ length: rows }, (_, i) => (
          <SkeletonBlock key={i} className="h-16 w-full" />
        ))}
      </div>
    </>
  );
  if (bare) {
    return <div role="status" aria-busy="true" aria-live="polite" data-testid="skeleton"><span className="sr-only">{label}</span>{body}</div>;
  }
  return <Frame label={label}><Header />{body}</Frame>;
}

/** The fields of a form (business data). */
export function FormSkeleton({ fields = 5, label = "Cargando…" }: { fields?: number; label?: string }) {
  return (
    <div role="status" aria-busy="true" aria-live="polite" data-testid="skeleton" className="bg-white border border-linea p-6 space-y-5">
      <span className="sr-only">{label}</span>
      {Array.from({ length: fields }, (_, i) => (
        <div key={i} className="space-y-2">
          <SkeletonBlock className="h-3 w-24" />
          <SkeletonBlock className="h-10 w-full" />
        </div>
      ))}
    </div>
  );
}
