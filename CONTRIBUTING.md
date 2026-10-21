# Contribuir

Un cambio economico debe explicar unidades, estado previo, transicion y
resultado esperado. Incluya un test Go para funciones del libro o un test de
integracion TypeScript cuando cambie la CLI o el JSON.

```bash
npm ci
bash scripts/ci.sh
```

Las modificaciones de campos JSON requieren actualizar `docs/integracion.md`
y `CHANGELOG.md`. Mantenga nombres de escenario, trazas y valores de activos
coherentes. No añada credenciales ni direcciones de infraestructura externa.
