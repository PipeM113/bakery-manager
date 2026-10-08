# bakery-manager

Herramienta interna de la pastelería: ingredientes, recetas, costos y precio sugerido, cotizaciones, ventas, gastos y analítica. Es además un caso de portafolio.

> Estado: reactivándose (etapa 0). Hoy no está desplegado. El plan está en [`docs/backlog.md`](docs/backlog.md) y la decisión de arquitectura en [`docs/adr/`](docs/adr/).

## Estructura

```
backend/   API en Go (chi, pgx), migraciones en backend/db/migrations
frontend/  React 19, Vite, Tailwind
docs/      ADR, fuentes verificadas y backlog
infra/     Postgres local solo para los tests
```

## Probar el backend

Los tests usan un Postgres local desechable y se niegan a correr contra cualquier otro servidor (por ejemplo Supabase). Sin `TEST_DATABASE_URL` los tests de base de datos se saltan en tu máquina; en CI su ausencia es un error.

```bash
docker compose -f infra/docker-compose.yml up -d --wait
export TEST_DATABASE_URL=postgres://postgres:postgres@127.0.0.1:54330/postgres
cd backend
go vet ./...
go test -count=1 ./...
```

Esas credenciales son solo de la base desechable de pruebas. Para apagarla: `docker compose -f infra/docker-compose.yml down -v`.

## Desarrollo local

Copia `backend/.env.example` a `backend/.env` y complétalo con valores tuyos. Nunca se sube al repositorio. Los secretos van solo en variables de entorno.

## Forma de trabajo

Una rama por sprint (`chore/sprint-N-...`), commits pequeños con conventional commits, y los tests de aceptación se congelan una vez aprobados. Los cambios a un test aprobado se piden y se explican.
