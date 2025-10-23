# Build Secret Mount Example

This example demonstrates how to use build secrets in Kaniko, allowing you to securely mount secrets during the build process without exposing them in the final image.

## Overview

Build secrets provide a secure way to use sensitive information (like API keys, credentials, certificates) during your container build without:
- Including them in the final image
- Exposing them in image history
- Committing them to your Dockerfile

## Usage

### Basic Usage

1. Create a secret file on your host:
```bash
echo "my-secret-value" > /path/to/secret.txt
```

2. Use the secret in your Dockerfile:
```dockerfile
RUN --mount=type=secret,id=mysecret \
    cat /run/secrets/mysecret
```

3. Build with Kaniko, providing the secret:
```bash
/kaniko/executor \
  --dockerfile=Dockerfile \
  --context=/workspace \
  --secret id=mysecret,src=/path/to/secret.txt \
  --destination=myregistry/myimage:latest
```

### Using Environment Variables as Secrets

You can also provide secrets from environment variables:

```bash
export MY_SECRET="secret-from-env"

/kaniko/executor \
  --dockerfile=Dockerfile \
  --context=/workspace \
  --secret id=mysecret,env=MY_SECRET \
  --destination=myregistry/myimage:latest
```

### Multiple Secrets

Provide multiple secrets by using the `--secret` flag multiple times:

```bash
/kaniko/executor \
  --dockerfile=Dockerfile \
  --context=/workspace \
  --secret id=secret1,src=/path/to/secret1.txt \
  --secret id=secret2,src=/path/to/secret2.txt \
  --secret id=secret3,env=SECRET_ENV_VAR \
  --destination=myregistry/myimage:latest
```

## Secret Mount Options

### In Dockerfile

When using `RUN --mount=type=secret`, you can specify:

- `id=<id>`: (Required) The ID of the secret to mount. Must match the ID provided to the executor.
- `target=<path>`: The path where the secret will be mounted. Default: `/run/secrets/<id>`
- `required=true|false`: Whether the build should fail if the secret is not provided. Default: `true`
- `mode=<octal>`: File permissions for the secret. Default: `0400` (read-only for owner)
- `uid=<uid>`: User ID for the secret file. Default: `0` (root)
- `gid=<gid>`: Group ID for the secret file. Default: `0` (root)

Example:
```dockerfile
RUN --mount=type=secret,id=apikey,target=/tmp/apikey,mode=0400,uid=1000 \
    use-api-key /tmp/apikey
```

### On Command Line

When running the executor, use the `--secret` flag:

- `id=<id>`: (Required) The ID of the secret
- `src=<path>`: Path to the secret file on the host
- `env=<var>`: Name of environment variable containing the secret

**Note**: You must specify either `src` or `env`, but not both.

Example:
```bash
--secret id=mysecret,src=/path/to/secret.txt
--secret id=apikey,env=API_KEY
```

## Security Features

1. **No Image Persistence**: Secrets are mounted temporarily during RUN commands and are automatically removed afterwards. They never become part of the image layers.

2. **Secure Cleanup**: When secrets are unmounted, the files are first overwritten with zeros before being deleted.

3. **Not in History**: Secret mount directives are stripped from the command history, so they don't appear in `docker history`.

4. **Proper Permissions**: Secrets are mounted with restrictive permissions (default 0400) to prevent unauthorized access.

## Common Use Cases

### 1. Private Package Installation

```dockerfile
FROM node:16

RUN --mount=type=secret,id=npmrc,target=/root/.npmrc \
    npm install private-package
```

```bash
/kaniko/executor \
  --secret id=npmrc,src=$HOME/.npmrc \
  --dockerfile=Dockerfile \
  --destination=myregistry/myapp:latest
```

### 2. Downloading from Private Repository

```dockerfile
FROM alpine:latest

RUN --mount=type=secret,id=github-token \
    apk add --no-cache git && \
    git clone https://$(cat /run/secrets/github-token)@github.com/private/repo.git
```

```bash
/kaniko/executor \
  --secret id=github-token,env=GITHUB_TOKEN \
  --dockerfile=Dockerfile \
  --destination=myregistry/myapp:latest
```

### 3. Building with Private Certificates

```dockerfile
FROM golang:1.19

RUN --mount=type=secret,id=ca-cert,target=/usr/local/share/ca-certificates/custom-ca.crt \
    update-ca-certificates && \
    go build -o myapp
```

```bash
/kaniko/executor \
  --secret id=ca-cert,src=/etc/ssl/certs/custom-ca.crt \
  --dockerfile=Dockerfile \
  --destination=myregistry/myapp:latest
```

## Kubernetes Example

When running Kaniko in Kubernetes, you can mount secrets from Kubernetes Secrets:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kaniko
spec:
  containers:
  - name: kaniko
    image: gcr.io/kaniko-project/executor:latest
    args:
    - "--dockerfile=Dockerfile"
    - "--context=git://github.com/myuser/myrepo.git"
    - "--secret=id=mysecret,src=/kaniko/secrets/secret.txt"
    - "--destination=myregistry/myimage:latest"
    volumeMounts:
    - name: secret-volume
      mountPath: /kaniko/secrets
      readOnly: true
  volumes:
  - name: secret-volume
    secret:
      secretName: my-kubernetes-secret
```

## Best Practices

1. **Use Required Secrets Carefully**: Set `required=false` only when the secret is truly optional for the build.

2. **Minimize Secret Scope**: Only mount secrets in the specific RUN commands that need them, not for the entire Dockerfile.

3. **Use Specific Targets**: Specify custom target paths to avoid conflicts when using multiple secrets.

4. **Don't Echo Secrets**: Avoid logging or echoing secret values in your build commands, as they may appear in build logs.

5. **Rotate Secrets Regularly**: Since secrets are provided at build time, make sure to rotate them regularly and rebuild images when secrets change.

## Troubleshooting

### Secret not found error

If you see an error like "secret with id X not found":
- Ensure the `id` in your Dockerfile matches the `id` in the `--secret` flag
- Verify the secret file exists at the specified `src` path
- Check that the environment variable is set if using `env`

### Permission denied errors

If you encounter permission issues:
- Verify the file permissions on the host secret file
- Check that the Kaniko executor has access to read the secret file
- Consider adjusting the `uid`, `gid`, and `mode` parameters in the Dockerfile

### Secrets appearing in logs

If you see secret values in logs:
- Avoid echoing or printing secret values in RUN commands
- Be cautious with verbose logging options during build
- Review your build scripts for accidental secret exposure

## Comparison with Docker BuildKit

Kaniko's secret mount implementation is compatible with Docker BuildKit's secret syntax:

```bash
# Docker BuildKit
docker build --secret id=mysecret,src=/path/to/secret .

# Kaniko
/kaniko/executor --secret id=mysecret,src=/path/to/secret --dockerfile=Dockerfile
```

This means Dockerfiles using secrets can work with both Docker BuildKit and Kaniko without modifications.

