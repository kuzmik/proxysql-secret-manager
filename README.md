# ProxySQL Secret Manager

### NB
This code is very specific for our use case at work, though maybe someone out there could find it useful. Also it turns out this wasn't really necessary, as I found it cleaner to handle the problem by creating some extra secrets in the proper format via Terrraform.

-----

This code is intended to run in an init container in the ProxySQL pod, to lay down the credential files in the filesystem before ProxySQL boots. It will fetch a secret values from GCSM (google cloud secret manager) and interpolate those values into some config file templates, then write those files to disk. ProxySQL can then read the files via the `@include` directive in the proxysql.cnf file.

The reason this exists is because the ProxySQL configuration requires a very specific format (it uses libconfig, specifically) when using `@include filename`, for example:

```
cluster_password = "whatever_the_password_is"
```

This makes using the existing secrets files impossible without extra formatting.

Up until recently we were manually modifying a secret with all of the config files in the specific format, which was labor intensive and error prone; it also meant a bunch of manual work was required before we could deploy proxysql to a new k8s namespace or env.
