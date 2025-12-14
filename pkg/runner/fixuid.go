package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
)

// UserMapping contains host user information for UID/GID mapping
type UserMapping struct {
	UID      int
	GID      int
	Username string
	HomeDir  string
}

// GetHostUser returns the current host user's UID/GID
func GetHostUser() (*UserMapping, error) {
	if runtime.GOOS == "windows" {
		// Windows doesn't use UID/GID, return defaults
		return &UserMapping{
			UID:      1000,
			GID:      1000,
			Username: os.Getenv("USERNAME"),
			HomeDir:  os.Getenv("USERPROFILE"),
		}, nil
	}

	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}

	uid, _ := strconv.Atoi(currentUser.Uid)
	gid, _ := strconv.Atoi(currentUser.Gid)

	return &UserMapping{
		UID:      uid,
		GID:      gid,
		Username: currentUser.Username,
		HomeDir:  currentUser.HomeDir,
	}, nil
}

// FixuidEntrypoint generates an entrypoint script that fixes UID/GID
const FixuidEntrypoint = `#!/bin/sh
# Container-Maker UID/GID Fix Entrypoint
# This script runs as root, fixes permissions, then drops to target user

set -e

TARGET_UID="${CM_TARGET_UID:-1000}"
TARGET_GID="${CM_TARGET_GID:-1000}"
TARGET_USER="${CM_TARGET_USER:-developer}"
WORKDIR="${CM_WORKDIR:-/workspace}"

# Skip if already running as target user
if [ "$(id -u)" = "$TARGET_UID" ]; then
    exec "$@"
fi

# Create group if it doesn't exist
if ! getent group "$TARGET_GID" > /dev/null 2>&1; then
    groupadd -g "$TARGET_GID" "$TARGET_USER" 2>/dev/null || true
fi

# Create user if it doesn't exist
if ! id -u "$TARGET_USER" > /dev/null 2>&1; then
    useradd -u "$TARGET_UID" -g "$TARGET_GID" -m -s /bin/sh "$TARGET_USER" 2>/dev/null || true
fi

# Fix UID if user exists but has wrong UID
CURRENT_UID=$(id -u "$TARGET_USER" 2>/dev/null || echo "0")
if [ "$CURRENT_UID" != "$TARGET_UID" ] && [ "$CURRENT_UID" != "0" ]; then
    usermod -u "$TARGET_UID" "$TARGET_USER" 2>/dev/null || true
fi

# Fix GID if needed
CURRENT_GID=$(id -g "$TARGET_USER" 2>/dev/null || echo "0")
if [ "$CURRENT_GID" != "$TARGET_GID" ] && [ "$CURRENT_GID" != "0" ]; then
    groupmod -g "$TARGET_GID" "$TARGET_USER" 2>/dev/null || true
    usermod -g "$TARGET_GID" "$TARGET_USER" 2>/dev/null || true
fi

# Fix ownership of workspace
if [ -d "$WORKDIR" ]; then
    chown -R "$TARGET_UID:$TARGET_GID" "$WORKDIR" 2>/dev/null || true
fi

# Fix home directory
if [ -d "/home/$TARGET_USER" ]; then
    chown -R "$TARGET_UID:$TARGET_GID" "/home/$TARGET_USER" 2>/dev/null || true
fi

# Drop privileges and execute command
# Try gosu first, then su-exec, then su
if command -v gosu > /dev/null 2>&1; then
    exec gosu "$TARGET_USER" "$@"
elif command -v su-exec > /dev/null 2>&1; then
    exec su-exec "$TARGET_USER" "$@"
else
    exec su -s /bin/sh "$TARGET_USER" -c "$*"
fi
`

// InjectFixuidEntrypoint injects the fixuid entrypoint into a running container
func InjectFixuidEntrypoint(ctx context.Context, containerID, backend string) error {
	// Write entrypoint script to container
	cmd := exec.CommandContext(ctx, backend, "exec", "-i", containerID, "sh", "-c",
		"cat > /tmp/cm-entrypoint.sh && chmod +x /tmp/cm-entrypoint.sh")
	cmd.Stdin = strings.NewReader(FixuidEntrypoint)
	return cmd.Run()
}

// GetUIDMappingEnv returns environment variables for UID/GID mapping
func GetUIDMappingEnv() []string {
	mapping, err := GetHostUser()
	if err != nil {
		return nil
	}

	return []string{
		fmt.Sprintf("CM_TARGET_UID=%d", mapping.UID),
		fmt.Sprintf("CM_TARGET_GID=%d", mapping.GID),
		fmt.Sprintf("CM_TARGET_USER=%s", mapping.Username),
	}
}

// ShouldUseUIDMapping determines if UID mapping should be used
func ShouldUseUIDMapping() bool {
	// Don't use UID mapping on Windows
	if runtime.GOOS == "windows" {
		return false
	}

	// Don't use UID mapping if already root
	if os.Geteuid() == 0 {
		return false
	}

	// Check if running in rootless Docker
	dockerHost := os.Getenv("DOCKER_HOST")
	if strings.Contains(dockerHost, "rootless") {
		return false // Rootless Docker handles UID mapping automatically
	}

	return true
}

// GetUserArg returns the --user argument for docker run
func GetUserArg() string {
	if !ShouldUseUIDMapping() {
		return ""
	}

	mapping, err := GetHostUser()
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%d:%d", mapping.UID, mapping.GID)
}
