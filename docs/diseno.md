# Diseño del manager

La app sigue el manual de marca de la web. La **fuente única** es
`angeles_bakery_web/docs/diseno.md` (y sus estilos `src/frontend/src/styles/global.css`); si ahí
cambia algo, se cambia aquí. Este archivo solo resume lo que aplica a la app de gestión.

## Colores

Definidos en `frontend/tailwind.config.js`.

| Token (clase) | Valor | Uso |
|---|---|---|
| `crema` | `#FAF6F1` | fondo de la app y de los campos |
| `lino` | `#E8DDD3` | fondos secundarios, filas, scroll |
| `salvia` | `#C4C9B8` | fondo de estados de éxito (solo fondo) |
| `rosa` | `#D9B8AE` | fondo de avisos y alertas (solo fondo) |
| `negro` | `#1C1C1C` | texto, botones principales; `negro/70` para texto secundario |
| `oro` | `#735523` | acento: títulos pequeños, enlaces, bordes al enfocar |
| `oro-claro` | `#EECC6A` | solo sobre fondos oscuros |
| `peligro` | `#8A1C1C` | errores y valores negativos |
| `linea` | `rgba(115, 85, 35, 0.35)` | bordes finos dorados |

## Reglas de contraste (WCAG)

- El texto es `negro` (15,8:1 sobre crema); el secundario, `negro/70`, también supera 4,5:1.
- El dorado en texto sobre fondo claro es siempre `oro` (6,4:1 sobre crema). El dorado de antes
  (`#B8860B`) tenía 3,1:1 y no cumplía.
- **Salvia y rosa son solo fondos**: sobre crema su contraste es de 1,6 a 1,7:1, así que nunca van
  como color de texto. `oro-claro` solo sobre fondos oscuros.
- Estados: éxito en fondo salvia con texto negro, aviso en fondo rosa, error en `peligro`
  (9,3:1 sobre blanco). Una ganancia positiva va en negro y una negativa en `peligro`.
- El foco visible es un contorno de 2 px en `oro`.

El test `frontend/tests/brand.test.mjs` calcula estos contrastes y revisa que no vuelva la paleta
anterior.

## Tipografía

Playfair Display para títulos y DM Sans Light para el texto, las mismas de la web, instaladas con
`@fontsource` (versión 5.3.0) para que no se pidan a un servidor externo.

## Elementos

- Botones principales: fondo `negro`, letras `crema`, mayúsculas con espacio entre letras.
- Líneas separadoras finas doradas (`oro` con transparencia) y mucho espacio.
- Logo: `frontend/src/assets/brand/logo-color.svg` (copiado de `angeles_bakery_web/assets/logos`).

## Pendiente

- (Resuelto en el sprint 12) El PDF de las cotizaciones usa `crema`, `oro` y `negro`, Playfair Display
  y DM Sans (incluidas en el servidor) y el logo en color. Ver `docs/cotizaciones.md`.
- Sinera, la tipografía de título del manual, se usará cuando haya licencia comercial (como en la
  web).

## Diseño responsivo

La app se usa sobre todo en **celular y en computador**. Las pantallas se adaptan, no se rediseñan.

**Anchos de referencia** (los que mide la suite): 360 y 390 px (celulares), 768 px (tablet) y 1280 px
(computador). Bajo 768 px es "celular" (el punto `md` de Tailwind): barra inferior, listas como
tarjetas. Desde 768 px hay barra lateral y las listas largas son tablas.

| Qué | Celular (menos de 768 px) | Desde 768 px |
|---|---|---|
| Navegación | Barra inferior con Dashboard, Ingredientes, Recetas, Ventas y **Más** (Costos, Costos fijos, Gastos, Rentabilidad) | Barra lateral |
| Ingredientes y cotizaciones | **Tarjetas** con los datos clave y sus acciones | Tabla |
| Botones, enlaces y campos | Al menos **44 px** de alto; los campos con texto de **16 px** (así el celular no hace zoom solo) | Los del diseño normal |
| Ventanas | Componente `Modal`: cabe en la pantalla, el cuerpo se desplaza y el botón principal queda siempre a mano, **por encima de la barra inferior** (capa `z-[60]`; la barra es `z-50`) | Igual, centradas |
| Cifras grandes y gráfico | La grilla pasa a una columna para que ninguna cifra se corte; el gráfico escala al ancho | Más columnas |

**Reglas globales** (en `src/index.css`): bajo 768 px todo `button`, `a[href]` y campo mide 44 px
como mínimo y los campos usan 16 px. Las ventanas nuevas deben usar `components/Modal.tsx` y marcar
su botón principal con `data-primary-action`.

**Marcas que las pantallas deben tener** (la suite las usa para medir): `data-testid="bottom-nav"` y
`"sidebar"`, `"nav-more"` y `"nav-more-menu"`, `"ingredients-list"` y `"quotations-list"` con
`data-view="cards" | "table"`, `data-card` en cada tarjeta, `role="dialog"`, `data-kpi` en las
cifras grandes y `data-testid="trend-chart"` en el gráfico.

**Cómo se prueba:** `npm run test:responsive` (en `frontend/`) abre un navegador real (Playwright)
con una API simulada, recorre las 9 pantallas en los cuatro anchos y revisa que no haya scroll
horizontal, que lo que se toca mida 44 px, que las ventanas quepan y que las cifras no se corten.
Deja una captura de cada pantalla y ancho en `frontend/test-results/responsive/`. Tarda varios
minutos; por eso `npm test` no la incluye.

**Lo que no hace:** instalar la app en el celular (PWA), modo oscuro, ni rediseñar pantallas.
