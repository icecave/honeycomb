# Changelog

## Unreleased

- **[FIX]** Generated certificates now include the issuer CA certificate in the chain (thanks [@koshatul])
- **[NEW]** Add support for parsing the client's remote address from [PROXY protocol](https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt) headers (thanks [@koshatul])
- **[NEW]** Add `CA_PATH` environment variable for specifying the location of CA bundles
- **[FIX]** `CA_PATH` did not affect connections to upstream services ([@koshatul])

## 0.3.3 (2017-03-16)

- **[NEW]** Add Comodo and GoDaddy intermediate certificates to the CA bundle (thanks [@koshatul])

## 0.3.2 (2017-03-13)

- **[NEW]** Allow specification of static routes via environment variables

## 0.3.1 (2017-03-10)

- **[FIX]** Send `X-Forwarded-Proto` and `X-Forwarded-SSL` headers

## 0.3.0 (2017-03-07)

- **[NEW]** Add support for multiple match labels

## 0.2.0 (2017-03-07)

- **[BC]** Remove support for loading certificates from S3, in favour of Docker secrets
- **[NEW]** Add file-based certificate provider
- **[FIXED]** Fix PKCS#8 private key loading
- **[IMPROVED]** Add HSTS headers

## 0.1.0 (2017-03-03)

- Initial release

[@koshatul]: https://github.com/koshatul
