# Integracion CLI y contrato JSON

El ejecutable expone cuatro comandos principales:

```text
riftdtl --list
riftdtl scenario <nombre>
riftdtl status <nombre>
riftdtl validate <nombre>
```

`--list` enumera escenarios disponibles. `scenario` devuelve el libro de
eventos y todas las vistas economicas; `status` entrega un subconjunto para
operaciones; `validate` comprueba el contrato estructural y devuelve el
digest del estado. Un escenario desconocido produce codigo de salida distinto
de cero y no emite JSON parcial.

## Ejemplo de uso

```bash
npm run build
./out/riftdtl scenario cross-shard > cross-shard.json
./out/riftdtl status cross-shard > status.json
./out/riftdtl validate cross-shard
```

La redireccion de salida es opcional. `status.json` incluye:

```json
{
  "protocol": "RiftDTL",
  "scenario": "cross-shard",
  "rounds": 1,
  "transfer_count": 1,
  "transfers": {"queued": 0, "finalized": 1, "failed": 0, "rejected": 0}
}
```

El fragmento omite `clock`, `state_digest`, `shards` y `assets`; el programa
los devuelve completos. Los importes son enteros y no deben transformarse a
`number` de JavaScript si una integracion futura puede exceder el rango de
enteros seguros.

## Reconciliacion del informe

Para cada activo, la suma de `shards[].available` coincide con
`assets[].available` del resumen operativo. La suite de integracion verifica
esta relacion. En el informe detallado, cada evento contiene un ID y marca de
reloj; conserve el digest cuando archive o compare informes.

## Evolucion del contrato

El JSON del release 1.0.0 usa `protocol` como identificador principal. Los
consumidores deben ignorar campos desconocidos y fallar ante la ausencia de
campos economicos obligatorios. Cambios de significado de campos requieren
version nueva; no se deben reinterpretar silenciosamente importes o estados.
