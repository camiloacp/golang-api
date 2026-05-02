# Generación de claves RSA para JWT

## Comandos

```bash
# Clave privada RSA 4096 bits
openssl genrsa -out private.pem 4096

# Clave pública derivada de la privada
openssl rsa -in private.pem -pubout -out public.pem
```

> Agrega `*.pem` al `.gitignore` — las claves nunca deben subir al repositorio.

---

## `openssl genrsa -out private.pem 4096`

**`openssl`** — CLI de OpenSSL, la librería criptográfica estándar.

**`genrsa`** — subcomando para generar una clave RSA. RSA es un algoritmo de criptografía asimétrica: genera un par de claves matemáticamente relacionadas (privada y pública). Lo que firma la privada, solo la pública puede verificarlo — y viceversa.

**`-out private.pem`** — escribe el resultado en `private.pem`. El formato PEM (Privacy Enhanced Mail) es texto en Base64 entre los delimitadores `-----BEGIN RSA PRIVATE KEY-----` y `-----END RSA PRIVATE KEY-----`.

**`4096`** — tamaño de la clave en bits. A mayor tamaño, más difícil de romper por fuerza bruta, pero más lento al firmar:

| Bits | Seguridad |
|------|-----------|
| 2048 | Mínimo aceptable hoy |
| 4096 | Estándar en banca y salud |
| 8192 | Prácticamente irrompible, pero muy lento |

Internamente, RSA genera dos números primos enormes aleatorios, los multiplica, y de esa operación deriva las claves. La seguridad se basa en que factorizar ese producto es computacionalmente inviable.

---

## `openssl rsa -in private.pem -pubout -out public.pem`

**`rsa`** — subcomando para operar sobre claves RSA existentes (a diferencia de `genrsa` que las crea).

**`-in private.pem`** — lee la clave privada generada en el paso anterior. La clave pública está matemáticamente embebida dentro de la privada, por eso se puede extraer de ahí.

**`-pubout`** — indica que la salida debe ser la clave pública. Sin este flag, por defecto reescribiría la privada. El formato de salida cambia a `-----BEGIN PUBLIC KEY-----`, que sigue el estándar PKIX/SubjectPublicKeyInfo, compatible con la mayoría de librerías JWT.

**`-out public.pem`** — escribe la clave pública en este archivo.

---

## Cómo se usan en JWT (RS256)

```
Firma:        payload + private.pem  →  token JWT
Verificación: token JWT + public.pem →  válido / inválido
```

El servidor firma con la privada. Cualquier cliente (u otro servicio) puede verificar con la pública sin poder falsificar un token nuevo, porque no tiene la privada.
