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

- El PDF de las cotizaciones aún no usa estos colores ni las tipografías: requiere incluir las
  fuentes en el servidor.
- Sinera, la tipografía de título del manual, se usará cuando haya licencia comercial (como en la
  web).
