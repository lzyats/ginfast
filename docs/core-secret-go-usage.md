# core-secret-go usage

The project now supports decrypting `ENC(...)` values during config load.

## How to use

1. Fill the `secret` section in `config/config.yml`.
2. Replace plaintext secrets with real `ENC(...)` ciphertexts.
3. Prefer setting `APP_SECRET` in the environment instead of storing `secret.app_secret` in the file.
4. Start the service normally. Config hot reload will also re-run secret decryption.

## Example targets

Typical sensitive fields in this project:

- `token.jwttokensignkey`
- `redis.password`
- `gormv2.mysql.write.pass`
- `gormv2.mysql.read.pass`
- `upload.qiniu_config.access_key`
- `upload.qiniu_config.secret_key`

See `config/config.secret.example.yml` for a ready-to-copy example layout.

## Important note

This SDK only provides decryption in the current repository integration.
It does not generate ciphertext locally here.
You need valid ciphertexts from your secret service before replacing live plaintext values.
