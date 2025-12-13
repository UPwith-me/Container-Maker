package runner

// EntrypointScript is a shell script that handles UID/GID mapping.
// It checks the ownership of the current directory (workspace) and creates a user
// with the same UID/GID if it doesn't exist. Respects CM_TARGET_USER if set.
const EntrypointScript = `#!/bin/sh
set -e

# If we are not root, just run the command
if [ "$(id -u)" != "0" ]; then
    exec "$@"
fi

# Check if a specific user is requested via CM_TARGET_USER
if [ -n "$CM_TARGET_USER" ]; then
    # Check if user exists
    if getent passwd "$CM_TARGET_USER" >/dev/null 2>&1; then
        USERNAME="$CM_TARGET_USER"
    else
        # Create the user if it doesn't exist
        adduser -D "$CM_TARGET_USER" 2>/dev/null || useradd -m "$CM_TARGET_USER" 2>/dev/null || true
        USERNAME="$CM_TARGET_USER"
    fi
else
    # Get the UID/GID of the current directory (workspace)
    TARGET_UID=$(stat -c "%u" . 2>/dev/null || stat -f "%u" .)
    TARGET_GID=$(stat -c "%g" . 2>/dev/null || stat -f "%g" .)
    
    # Check if a user with TARGET_UID already exists
    if ! getent passwd "$TARGET_UID" >/dev/null 2>&1; then
        # Create group if it doesn't exist
        if ! getent group "$TARGET_GID" >/dev/null 2>&1; then
            addgroup -g "$TARGET_GID" cm_group 2>/dev/null || groupadd -g "$TARGET_GID" cm_group 2>/dev/null || true
        fi
        
        # Get group name
        GROUPNAME=$(getent group "$TARGET_GID" | cut -d: -f1)
        
        # Create user
        adduser -u "$TARGET_UID" -G "$GROUPNAME" -D cm_user 2>/dev/null || \
        useradd -u "$TARGET_UID" -g "$TARGET_GID" -m cm_user 2>/dev/null || true
    fi

    # Get the username for the UID
    USERNAME=$(getent passwd "$TARGET_UID" | cut -d: -f1)
fi

# If we still don't have a valid username, run as root
if [ -z "$USERNAME" ]; then
    exec "$@"
fi

# Execute the command as the user
# Try su-exec, then gosu, then su
if command -v su-exec >/dev/null 2>&1; then
    exec su-exec "$USERNAME" "$@"
elif command -v gosu >/dev/null 2>&1; then
    exec gosu "$USERNAME" "$@"
else
    exec su "$USERNAME" -c "$*"
fi
`

// GetEntrypoint returns the entrypoint script content
func GetEntrypoint() string {
	return EntrypointScript
}
