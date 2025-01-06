# Kubernetes Secret Generator

[![REUSE status](https://api.reuse.software/badge/github.com/SAP/secret-generator)](https://api.reuse.software/info/github.com/SAP/secret-generator)

## About this project

This repository contains a [Mutating Admission Webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers) for Kubernetes secrets that allows to generate certain secret values (e.g. passwords) upon first appearance of the according secret key. For example:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  labels:
    secret-generator.cs.sap.com/enabled: "true"
stringData:
  my-password: "%generate:password:length=16"
  my-uuid: "%generate:uuid"
  my-other-key: "some static value"
```

To make it clear, the generation of a value only happens if the according key is not present in the secret. Existing values will never be touched (even if the `%generate` clause changes).

By default - when using the [Helm chart](https://github.com/sap/secret-generator-helm) - the webhook is called for secrets having the label `secret-generator.cs.sap.com/enabled: "true"`, but this can be overridden in the chart's configuration.

Then, secret values of the form `%generate:<type>[:<arg=value>;<arg=value>;...]` will be replaced accordingly.
Currently, two generator types are supported: `uuid` and `password`:
- `uuid` will generate a [RFC4122](https://datatracker.ietf.org/doc/html/rfc4122) UUIDv4 and allows the following arguments:
  - `encoding=<base32|base64|base64_url|base64_raw|base64_raw_url>`: encoding to be applied to the generated uuid (note: use raw for no padding)
- `password` allows the following arguments:
  - `length=<1-99>`: length of the generated password (default 32)
  - `num_digits=<0-99>`: number of digits (0-9) in the generated password (default length/4)
  - `num_symbols=<0-99>`: number of symbols in the generated pasasword (default length/4)
  - `symbols=<chars>`: symbols (i.e. non-alphanumerics) to be used in the generated password (default: `~!@#$%^&*()_+-={}|:<>?,./`)
  - `encoding=<base32|base64|base64_url|base64_raw|base64_raw_url>`: encoding to be applied to the generated password (note: the actual length will be larger than specified by length then).

As a short form it is possible to just specify `%generate` as secret value, in which case a (32 character) password will be generated.

**Command line flags**

|Flag                         |Optional|Default|Description                                                 |
|-----------------------------|--------|-------|------------------------------------------------------------|
|--bind-address string         |yes     |:2443  |Webhook bind address                                        |
|--tls-key-file                |no      |-      |File containing the TLS private key used for SSL termination|
|--tls-cert-file               |no      |-      |File containing the TLS certificate matching the private key|

**References**

- Password generation uses [github.com/sethvargo/go-password/password](https://pkg.go.dev/github.com/sethvargo/go-password)

- UUID generation uses [github.com/google/uuid](https://pkg.go.dev/github.com/google/uuid)

## Requirements and Setup

The recommended deployment method is to use the [Helm chart](https://github.com/sap/secret-generator-helm):

```bash
helm upgrade -i secret-generator oci://ghcr.io/sap/secret-generator-helm/secret-generator
```

The API reference is here: [https://pkg.go.dev/github.com/sap/secret-generator](https://pkg.go.dev/github.com/sap/secret-generator).

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/SAP/secret-generator/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2025 SAP SE or an SAP affiliate company and secret-generator contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/SAP/secret-generator).
