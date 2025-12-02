#!/bin/bash
# Test script for pipeboard SSH peer functionality using Docker containers

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}==>${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

# Cleanup function
cleanup() {
    log "Cleaning up containers and network..."
    docker rm -f pipeboard-sender pipeboard-receiver 2>/dev/null || true
    docker network rm pipeboard-test 2>/dev/null || true
}

# Set trap for cleanup
trap cleanup EXIT

# Start fresh
cleanup

log "Creating Docker network..."
docker network create pipeboard-test

log "Starting Alpine containers..."
docker run -d --name pipeboard-sender --network pipeboard-test \
    -v "$PWD:/src:ro" \
    alpine:latest sleep infinity

docker run -d --name pipeboard-receiver --network pipeboard-test \
    -v "$PWD:/src:ro" \
    alpine:latest sleep infinity

log "Installing dependencies on both containers..."
for container in pipeboard-sender pipeboard-receiver; do
    docker exec $container sh -c '
        apk add --no-cache openssh go git xclip xvfb >/dev/null 2>&1
        ssh-keygen -A >/dev/null 2>&1
        echo "PermitRootLogin yes" >> /etc/ssh/sshd_config
        echo "PasswordAuthentication no" >> /etc/ssh/sshd_config
        echo "AcceptEnv DISPLAY" >> /etc/ssh/sshd_config
        /usr/sbin/sshd
        # Start virtual X server for xclip
        Xvfb :99 -screen 0 1024x768x24 >/dev/null 2>&1 &
        export DISPLAY=:99
        echo "export DISPLAY=:99" >> /root/.profile
        echo "export DISPLAY=:99" >> /root/.bashrc

        # Create xclip wrapper that forks to background
        # xclip waits for clipboard requests - fork it so pipeboard does not hang
        mv /usr/bin/xclip /usr/bin/xclip-real
        cat > /usr/bin/xclip << "EOF"
#!/bin/sh
# Wrapper for xclip that forks to background for copy operations
# Check if this is a copy operation (no -o flag)
if echo "$@" | grep -q "\-o"; then
    # Paste operation - run normally
    exec /usr/bin/xclip-real "$@"
else
    # Copy operation - save stdin to temp file, then run xclip in background
    TMPFILE=$(mktemp)
    cat > "$TMPFILE"
    cat "$TMPFILE" | /usr/bin/xclip-real "$@" >/dev/null 2>&1 &
    # Wait briefly for xclip to read the file
    sleep 0.1
    rm -f "$TMPFILE"
fi
EOF
        chmod +x /usr/bin/xclip
    '
done

log "Setting up SSH keys (sender -> receiver)..."
docker exec pipeboard-sender ssh-keygen -t ed25519 -f /root/.ssh/id_ed25519 -N "" >/dev/null 2>&1
PUBKEY_SENDER=$(docker exec pipeboard-sender cat /root/.ssh/id_ed25519.pub)
docker exec pipeboard-receiver sh -c "mkdir -p /root/.ssh && echo '$PUBKEY_SENDER' >> /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys"

log "Setting up SSH keys (receiver -> sender)..."
docker exec pipeboard-receiver ssh-keygen -t ed25519 -f /root/.ssh/id_ed25519 -N "" >/dev/null 2>&1
PUBKEY_RECEIVER=$(docker exec pipeboard-receiver cat /root/.ssh/id_ed25519.pub)
docker exec pipeboard-sender sh -c "echo '$PUBKEY_RECEIVER' >> /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys"

log "Adding receiver to sender's known_hosts..."
docker exec pipeboard-sender sh -c '
    ssh-keyscan -H pipeboard-receiver >> /root/.ssh/known_hosts 2>/dev/null
'

log "Adding sender to receiver's known_hosts..."
docker exec pipeboard-receiver sh -c '
    ssh-keyscan -H pipeboard-sender >> /root/.ssh/known_hosts 2>/dev/null
'

log "Building pipeboard on both containers..."
for container in pipeboard-sender pipeboard-receiver; do
    echo "  Building on $container..."
    if ! docker exec -w /src -e DISPLAY=:99 $container go build -buildvcs=false -o /usr/local/bin/pipeboard . 2>&1; then
        error "Build failed on $container"
        exit 1
    fi
done
success "Build completed on both containers"

log "Creating pipeboard config on sender..."
docker exec pipeboard-sender sh -c 'mkdir -p /root/.config/pipeboard /root/.ssh'
docker exec pipeboard-sender sh -c 'cat > /root/.config/pipeboard/config.yaml << EOF
version: 1
peers:
  receiver:
    ssh: root@pipeboard-receiver
    remote_cmd: env DISPLAY=:99 pipeboard
    default: true
EOF'
docker exec pipeboard-sender sh -c 'echo "SendEnv DISPLAY" >> /root/.ssh/config'

log "Creating pipeboard config on receiver..."
docker exec pipeboard-receiver sh -c 'mkdir -p /root/.config/pipeboard /root/.ssh'
docker exec pipeboard-receiver sh -c 'cat > /root/.config/pipeboard/config.yaml << EOF
version: 1
peers:
  sender:
    ssh: root@pipeboard-sender
    remote_cmd: env DISPLAY=:99 pipeboard
    default: true
EOF'
docker exec pipeboard-receiver sh -c 'echo "SendEnv DISPLAY" >> /root/.ssh/config'

echo ""
log "Running SSH connection test..."
if docker exec pipeboard-sender ssh -o StrictHostKeyChecking=no root@pipeboard-receiver echo "SSH works" >/dev/null 2>&1; then
    success "SSH connection works"
else
    error "SSH connection failed"
    exit 1
fi

echo ""
log "Test 1: Send text from sender to receiver"
docker exec -e DISPLAY=:99 pipeboard-sender sh -c 'echo "Hello from sender" | pipeboard copy'
docker exec -e DISPLAY=:99 pipeboard-sender pipeboard send receiver

sleep 1

RESULT=$(docker exec -e DISPLAY=:99 pipeboard-receiver pipeboard paste 2>/dev/null)
if [ "$RESULT" = "Hello from sender" ]; then
    success "Text sent and received successfully"
else
    error "Text transfer failed. Expected 'Hello from sender', got '$RESULT'"
    exit 1
fi

echo ""
log "Test 2: Send multiline content"
docker exec -e DISPLAY=:99 pipeboard-sender sh -c 'printf "Line 1\nLine 2\nLine 3" | pipeboard copy'
docker exec -e DISPLAY=:99 pipeboard-sender pipeboard send receiver

sleep 1

RESULT=$(docker exec -e DISPLAY=:99 pipeboard-receiver pipeboard paste 2>/dev/null)
if echo "$RESULT" | grep -q "Line 1" && echo "$RESULT" | grep -q "Line 3"; then
    success "Multiline content sent successfully"
else
    error "Multiline transfer failed"
    exit 1
fi

echo ""
log "Test 3: Bidirectional transfer (receiver -> sender)"
docker exec -e DISPLAY=:99 pipeboard-receiver sh -c 'echo "Reply from receiver" | pipeboard copy'
docker exec -e DISPLAY=:99 pipeboard-receiver pipeboard send sender

sleep 1

RESULT=$(docker exec -e DISPLAY=:99 pipeboard-sender pipeboard paste 2>/dev/null)
if [ "$RESULT" = "Reply from receiver" ]; then
    success "Bidirectional transfer works"
else
    error "Reverse transfer failed. Got '$RESULT'"
    exit 1
fi

echo ""
success "All SSH peer tests passed!"

echo ""
log "Test environment still running. Press Enter to cleanup..."
echo ""
log "Press Enter to cleanup and exit, or Ctrl+C to keep running..."
read -r

exit 0
