# Plan de Mejoras - Servicio de Pagos (Payment Service)

Este archivo detalla las tareas de refactorización y corrección priorizadas de acuerdo a su impacto en la estabilidad, resiliencia y seguridad del sistema.

---

## [ ] Prioridad 1: Correcciones Críticas (Bugs de Estabilidad)

### 1.1 Corregir llamada duplicada a `ListenAndServe` y lógica de error
* **Ubicación:** [main.go](file:///Users/jcmexdev/projects/golang/payment-service/cmd/payments-api/main.go)
* **Descripción:** Actualmente se inicia el servidor HTTP dos veces, provocando un error de puerto ocupado en los logs. Además, se detecta de forma incorrecta el cierre limpio del servidor.
* **Acciones:**
  - [ ] Eliminar la segunda llamada síncrona a `ListenAndServe`.
  - [ ] Corregir la validación de errores en el canal `serverErrors` para que use `!errors.Is(err, http.ErrServerClosed)`.

---

## [ ] Prioridad 2: Resiliencia del Flujo (Evitar pérdidas de datos)

### 2.1 Desasociar Contexto al Guardar la Respuesta
* **Ubicación:** [idempotency.go](file:///Users/jcmexdev/projects/golang/payment-service/internal/infra/http/middleware/idempotency.go)
* **Descripción:** Si el cliente desconecta la red abruptamente justo al finalizar la petición, `r.Context()` se cancela. Esto aborta el guardado en SQLite/Redis.
* **Acciones:**
  - [ ] Usar `context.WithoutCancel(r.Context())` antes de llamar a `m.repo.Save`.

---

## [ ] Prioridad 3: Robustez de la Idempotencia (Manejo de Errores de Servidor)

### 3.1 Liberar la llave de Idempotencia en Errores 5xx
* **Ubicación:** Varios archivos
* **Descripción:** Si tu servidor de pagos falla (código >= 500), la llave no debe quedarse bloqueada como `"processing"`. Debe eliminarse inmediatamente para permitir que el cliente intente de nuevo con la misma clave.
* **Acciones:**
  - [ ] Añadir método `Delete(ctx context.Context, key string) error` a la interfaz `IdempotencyRepository`.
  - [ ] Implementar `Delete` en el repositorio de **Redis** (`redis/idempotency.go`).
  - [ ] Implementar `Delete` en el repositorio de **SQLite** (`sqlite/idempotency.go`).
  - [ ] Implementar la delegación del método en `PersistenceCache` (`cache/persistence_cache.go`).
  - [ ] Invocar `m.repo.Delete` en el middleware cuando la respuesta tenga un estado `>= 500`.

---

## [ ] Prioridad 4: Optimización del Ciclo de Vida (Dual TTL)

### 4.1 Implementar Tiempos de Expiración Diferenciados (Lease Time)
* **Ubicación:** [PersistenceCache](file:///Users/jcmexdev/projects/golang/payment-service/internal/infra/cache/persistence_cache.go) y Middleware
* **Descripción:** No usar el TTL largo (ej: 1 hora) durante la fase de bloqueo. Si el servidor se apaga, la clave queda bloqueada demasiado tiempo.
* **Acciones:**
  - [ ] Establecer un TTL corto de bloqueo (ej: 2 a 5 minutos) en el método `Lock` para actuar como *Lease*.
  - [ ] Mantener el TTL largo configurado por el usuario en el método `Save`.

---

## [ ] Prioridad 5: Seguridad y Consistencia (Prevención de Colisiones)

### 5.1 Validar coincidencia del Cuerpo de la Petición (Request Body)
* **Ubicación:** Middleware y Estructura de Datos
* **Descripción:** Evitar que se envíen diferentes montos o cuentas de destino usando la misma clave de idempotencia erróneamente.
* **Acciones:**
  - [ ] Generar un hash (ej: SHA-256) del cuerpo de la petición en el middleware.
  - [ ] Guardar el hash junto con el registro en la base de datos.
  - [ ] Al recibir una llave duplicada, verificar que el hash del cuerpo actual coincida con el guardado; si no, devolver un error `400 Bad Request`.


ahora agrega alguna funcionalidad para que mi db o mis servicios fallen intencionalmente con algun atributo, quiero poder entender como funcionan los traces y como la observabilidad me ayuda a encontrar problemas pero para eso necesito generar cargas masivas con k8 y hacer que mi servici falle aveces y de ahi ir diagnosticando que paso ayudame con eso
- verificar account antes de insertar el payment 