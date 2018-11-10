## SSL Certificates

**For production:** Use an authentic cert-chain + key provided by whatever vendor you choose.
- lets-encrypt provides this service for free.

**For testing/development:** Use `openssl` to create a self-signed cert for `localhost` (credit
[letsencrypt](https://letsencrypt.org/docs/certificates-for-localhost/#making-and-trusting-your-own-certificates)):
```
openssl req -x509 -out localhost.crt -keyout localhost.key \
  -newkey rsa:2048 -nodes -sha256 \
  -subj '/CN=localhost' -extensions EXT -config <( \
   printf "[dn]\nCN=localhost\n[req]\ndistinguished_name = dn\n[EXT]\nsubjectAltName=DNS:localhost\nkeyUsage=digitalSignature\nextendedKeyUsage=serverAuth")
```

