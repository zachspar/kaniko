FROM alpine:latest

# Example 1: Using a secret in a RUN command
# The secret will be mounted at /run/secrets/mysecret by default
RUN --mount=type=secret,id=mysecret \
    cat /run/secrets/mysecret && \
    echo "Secret accessed successfully"

# Example 2: Using a secret with a custom target path
RUN --mount=type=secret,id=apikey,target=/tmp/apikey \
    cat /tmp/apikey > /dev/null && \
    echo "API key validated"

# Example 3: Using a secret with custom permissions
RUN --mount=type=secret,id=credentials,mode=0400,uid=1000,gid=1000 \
    echo "Credentials mounted with custom permissions"

# Example 4: Using an optional secret (won't fail if not provided)
RUN --mount=type=secret,id=optional-secret,required=false \
    if [ -f /run/secrets/optional-secret ]; then \
        echo "Optional secret found"; \
    else \
        echo "Optional secret not provided, continuing..."; \
    fi

# Example 5: Using multiple secrets in one RUN command
RUN --mount=type=secret,id=secret1 \
    --mount=type=secret,id=secret2 \
    cat /run/secrets/secret1 /run/secrets/secret2 > /dev/null && \
    echo "Multiple secrets accessed"

# The secrets are automatically removed after each RUN command completes
# They will not be present in the final image

WORKDIR /app
CMD ["/bin/sh"]

