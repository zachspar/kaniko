#!/bin/bash
# Example script demonstrating how to build with secrets in Kaniko

set -e

# Create example secrets
echo "my-secret-password" > /tmp/kaniko-secret1.txt
echo "api-key-12345" > /tmp/kaniko-secret2.txt
export MY_ENV_SECRET="secret-from-environment"

echo "Building with Kaniko secret mounts..."
echo "This example shows how to use secrets during build"
echo ""

# Example 1: Basic secret from file
echo "Example 1: Using secret from file"
/kaniko/executor \
  --dockerfile=examples/secret-mount-example/Dockerfile \
  --context=. \
  --secret id=mysecret,src=/tmp/kaniko-secret1.txt \
  --no-push \
  --tarPath=/tmp/image.tar

# Example 2: Multiple secrets (file + environment)
echo ""
echo "Example 2: Using multiple secrets (file + environment)"
/kaniko/executor \
  --dockerfile=examples/secret-mount-example/Dockerfile \
  --context=. \
  --secret id=mysecret,src=/tmp/kaniko-secret1.txt \
  --secret id=apikey,src=/tmp/kaniko-secret2.txt \
  --secret id=envsecret,env=MY_ENV_SECRET \
  --no-push \
  --tarPath=/tmp/image.tar

# Cleanup
rm -f /tmp/kaniko-secret1.txt /tmp/kaniko-secret2.txt

echo ""
echo "Build completed successfully!"
echo ""
echo "Key points:"
echo "- Secrets were available during build"
echo "- Secrets are NOT in the final image"
echo "- Secrets are NOT in the image history"
echo "- Check 'docker history <image>' to verify"

