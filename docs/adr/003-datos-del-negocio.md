# ADR-003: los datos del negocio se reparten entre el código, la base y las variables de entorno

- **Estado:** Propuesto (borrador de Claude, 2026-11-01; recoge lo decidido por Felipe en los sprints 12 y 13)
- **Fecha:** 2026-11-01

## Contexto

El PDF de cotización lleva datos del negocio: nombre, Instagram, teléfono, datos bancarios, RUT y correo. Algunos son la marca y no cambian; otros cambian (teléfono, cuenta); otros son sensibles (RUT y número de cuenta). Al principio se quiso que todo se editara en la app.

## Opciones consideradas

| Opción | Efecto | Problema |
|---|---|---|
| A. Todo editable en la base | Una sola pantalla | Nombre e Instagram no deberían editarse (son la marca); RUT y cuenta quedan en una tabla expuesta a cualquier cuenta |
| B. Todo en variables de entorno | Nada que editar en la app | Cambiar un teléfono exige entrar a Vercel y redesplegar |
| C. Repartido por naturaleza del dato | Cada dato vive donde corresponde | Tres lugares; hay que documentarlo |

## Decisión

Opción C.

- **Marca, en el código:** nombre ("Angeles Bakery") e Instagram, como constantes (`BrandName`, `BrandInstagramURL`). Cambiarlas es un cambio de código.
- **Editables, en la base:** teléfono, banco, tipo de cuenta, titular y correo, en `business_profile` (una sola fila), por `PUT /business`. El endpoint **rechaza campos desconocidos** (`DisallowUnknownFields`), no los ignora.
- **Sensibles, en el entorno:** `BIZ_RUT` y `BIZ_ACCOUNT`, que se leen al arrancar. Cambiarlos exige redesplegar.

## Razón

- Lo que es marca no debe poder cambiarse por error desde la app (decisión de Felipe, sprint 13).
- Lo que cambia seguido lo edita la dueña sin pasar por Vercel.
- El RUT y la cuenta no deben depender de que una cuenta de la app los edite (decisión de Felipe).

## Supuestos

| Supuesto | Estado |
|---|---|
| El PDF se genera a partir de lo guardado en la cotización y de estos tres orígenes | Confirmado |
| Dos cuentas de confianza editan `business_profile` | Confirmado (ADR-002) |
| Las variables de entorno no cambian a menudo | Inferido |

## Consecuencias

- Hay que redesplegar para cambiar el RUT o la cuenta (documentado en `docs/cotizaciones.md`).
- La configuración **no arranca** si faltan o son inválidas (`internal/config`).
- Un cuarto dato nuevo obliga a elegir su lugar con este criterio.

## Revisar si

- Se necesita cambiar el RUT o la cuenta con frecuencia.
- Hay más de un negocio o más de una marca.
- Llega la etapa 1 y la identidad pasa a Supabase Auth.
