#!/bin/bash
# pipeboard Demo - Try pipeboard without installing
# Usage: ./scripts/demo.sh

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

log() { echo -e "${BLUE}==>${NC} $1"; }
success() { echo -e "${GREEN}✓${NC} $1"; }
error() { echo -e "${RED}✗${NC} $1"; exit 1; }

# Cleanup function
cleanup() {
    echo ""
    log "Cleaning up demo environment..."
    docker rm -f pipeboard-demo-a pipeboard-demo-b 2>/dev/null || true
    docker network rm pipeboard-demo 2>/dev/null || true
    echo -e "${GREEN}Demo cleaned up. Thanks for trying pipeboard!${NC}"
}

show_banner() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}              ${BOLD}pipeboard Demo Environment${NC}                     ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}     The programmable clipboard router for terminals         ${CYAN}║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# Check for Docker
if ! command -v docker &> /dev/null; then
    error "Docker is required. Install from https://docker.com"
fi

if ! docker info &> /dev/null; then
    error "Docker daemon is not running"
fi

show_banner

echo -e "${YELLOW}This demo creates two connected containers to try pipeboard features:${NC}"
echo "  • Copy/paste between machines via SSH"
echo "  • Clipboard history and recall"
echo "  • Transform pipelines"
echo ""
echo -e "${BOLD}Requirements:${NC} Docker (~500MB for Alpine + Go)"
echo ""
read -p "Press Enter to start the demo, or Ctrl+C to cancel..."

# Cleanup any existing demo
docker rm -f pipeboard-demo-a pipeboard-demo-b 2>/dev/null || true
docker network rm pipeboard-demo 2>/dev/null || true

log "Creating demo network..."
docker network create pipeboard-demo

log "Starting containers (this takes 1-2 minutes on first run)..."
docker run -d --name pipeboard-demo-a --network pipeboard-demo \
    --hostname machine-a \
    -v "$PWD:/src:ro" \
    alpine:latest sleep infinity

docker run -d --name pipeboard-demo-b --network pipeboard-demo \
    --hostname machine-b \
    -v "$PWD:/src:ro" \
    alpine:latest sleep infinity

log "Installing dependencies..."
for container in pipeboard-demo-a pipeboard-demo-b; do
    docker exec $container sh -c '
        apk add --no-cache openssh go git xclip xvfb bash >/dev/null 2>&1
        ssh-keygen -A >/dev/null 2>&1
        echo "PermitRootLogin yes" >> /etc/ssh/sshd_config
        echo "PasswordAuthentication no" >> /etc/ssh/sshd_config
        echo "AcceptEnv DISPLAY" >> /etc/ssh/sshd_config
        /usr/sbin/sshd
        Xvfb :99 -screen 0 1024x768x24 >/dev/null 2>&1 &
        echo "export DISPLAY=:99" >> /root/.bashrc
        echo "export PS1=\"\[\033[1;32m\]\h\[\033[0m\]:\[\033[1;34m\]\w\[\033[0m\]\\$ \"" >> /root/.bashrc

        # xclip wrapper for headless environment
        mv /usr/bin/xclip /usr/bin/xclip-real
        cat > /usr/bin/xclip << "WRAPPER"
#!/bin/sh
if echo "$@" | grep -q "\-o"; then
    exec /usr/bin/xclip-real "$@"
else
    TMPFILE=$(mktemp)
    cat > "$TMPFILE"
    cat "$TMPFILE" | /usr/bin/xclip-real "$@" >/dev/null 2>&1 &
    sleep 0.1
    rm -f "$TMPFILE"
fi
WRAPPER
        chmod +x /usr/bin/xclip
    ' 2>/dev/null
done
success "Dependencies installed"

log "Setting up SSH between containers..."
# Generate and exchange keys
docker exec pipeboard-demo-a ssh-keygen -t ed25519 -f /root/.ssh/id_ed25519 -N "" >/dev/null 2>&1
docker exec pipeboard-demo-b ssh-keygen -t ed25519 -f /root/.ssh/id_ed25519 -N "" >/dev/null 2>&1

PUBKEY_A=$(docker exec pipeboard-demo-a cat /root/.ssh/id_ed25519.pub)
PUBKEY_B=$(docker exec pipeboard-demo-b cat /root/.ssh/id_ed25519.pub)

docker exec pipeboard-demo-a sh -c "echo '$PUBKEY_B' >> /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys"
docker exec pipeboard-demo-b sh -c "echo '$PUBKEY_A' >> /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys"

docker exec pipeboard-demo-a sh -c 'ssh-keyscan -H pipeboard-demo-b >> /root/.ssh/known_hosts 2>/dev/null'
docker exec pipeboard-demo-b sh -c 'ssh-keyscan -H pipeboard-demo-a >> /root/.ssh/known_hosts 2>/dev/null'
success "SSH configured"

log "Building pipeboard..."
for container in pipeboard-demo-a pipeboard-demo-b; do
    docker exec -w /src $container go build -buildvcs=false -o /usr/local/bin/pipeboard . 2>/dev/null
done
success "pipeboard built"

log "Configuring pipeboard..."
# Machine A config
docker exec pipeboard-demo-a sh -c 'mkdir -p /root/.config/pipeboard && cat > /root/.config/pipeboard/config.yaml << EOF
version: 1
defaults:
  peer: machine-b
peers:
  machine-b:
    ssh: root@pipeboard-demo-b
    remote_cmd: env DISPLAY=:99 pipeboard
fx:
  upper:
    shell: "tr a-z A-Z"
  lower:
    shell: "tr A-Z a-z"
  trim:
    shell: "sed \"s/^[[:space:]]*//;s/[[:space:]]*$//\""
  reverse:
    shell: "rev"
EOF'

# Machine B config
docker exec pipeboard-demo-b sh -c 'mkdir -p /root/.config/pipeboard && cat > /root/.config/pipeboard/config.yaml << EOF
version: 1
defaults:
  peer: machine-a
peers:
  machine-a:
    ssh: root@pipeboard-demo-a
    remote_cmd: env DISPLAY=:99 pipeboard
fx:
  upper:
    shell: "tr a-z A-Z"
  lower:
    shell: "tr A-Z a-z"
  trim:
    shell: "sed \"s/^[[:space:]]*//;s/[[:space:]]*$//\""
  reverse:
    shell: "rev"
EOF'
success "Configuration complete"

# Create welcome script
docker exec pipeboard-demo-a sh -c 'cat > /root/welcome.sh << "EOF"
#!/bin/bash
clear
echo ""
echo -e "\033[1;36m╔══════════════════════════════════════════════════════════════╗\033[0m"
echo -e "\033[1;36m║\033[0m  Welcome to \033[1mpipeboard\033[0m Demo - You are on \033[1;32mMachine A\033[0m            \033[1;36m║\033[0m"
echo -e "\033[1;36m╚══════════════════════════════════════════════════════════════╝\033[0m"
echo ""
echo -e "\033[1;33m📋 TRY THESE COMMANDS:\033[0m"
echo ""
echo -e "\033[1mBasic clipboard:\033[0m"
echo "  echo \"hello world\" | pipeboard       # copy to clipboard"
echo "  pipeboard paste                       # paste from clipboard"
echo ""
echo -e "\033[1mSend to Machine B:\033[0m"
echo "  echo \"secret message\" | pipeboard    # copy something"
echo "  pipeboard send                        # send to machine-b"
echo ""
echo -e "\033[1mTransforms:\033[0m"
echo "  echo \"hello\" | pipeboard && pipeboard fx upper && pipeboard paste"
echo "  pipeboard fx --list                   # see available transforms"
echo ""
echo -e "\033[1mHistory:\033[0m"
echo "  pipeboard history --local             # see clipboard history"
echo "  pipeboard recall 1                    # restore previous entry"
echo ""
echo -e "\033[1;34m💡 Open another terminal and run:\033[0m"
echo "  docker exec -it pipeboard-demo-b bash"
echo "  Then: pipeboard paste    # to see what Machine A sent!"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo ""
EOF
chmod +x /root/welcome.sh
echo "source /root/welcome.sh" >> /root/.bashrc'

docker exec pipeboard-demo-b sh -c 'cat > /root/welcome.sh << "EOF"
#!/bin/bash
clear
echo ""
echo -e "\033[1;36m╔══════════════════════════════════════════════════════════════╗\033[0m"
echo -e "\033[1;36m║\033[0m  Welcome to \033[1mpipeboard\033[0m Demo - You are on \033[1;33mMachine B\033[0m            \033[1;36m║\033[0m"
echo -e "\033[1;36m╚══════════════════════════════════════════════════════════════╝\033[0m"
echo ""
echo -e "\033[1;33m📋 TRY THESE COMMANDS:\033[0m"
echo ""
echo -e "\033[1mReceive from Machine A:\033[0m"
echo "  pipeboard paste                       # see what Machine A sent"
echo "  pipeboard recv                        # pull clipboard from A"
echo ""
echo -e "\033[1mSend back to Machine A:\033[0m"
echo "  echo \"reply from B\" | pipeboard"
echo "  pipeboard send                        # send to machine-a"
echo ""
echo -e "\033[1mPeek without copying:\033[0m"
echo "  pipeboard peek                        # view Machine A clipboard"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo ""
EOF
chmod +x /root/welcome.sh
echo "source /root/welcome.sh" >> /root/.bashrc'

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║${NC}                    ${BOLD}Demo Ready!${NC}                               ${GREEN}║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "You now have two machines connected via SSH:"
echo -e "  ${BOLD}Machine A:${NC} docker exec -it pipeboard-demo-a bash"
echo -e "  ${BOLD}Machine B:${NC} docker exec -it pipeboard-demo-b bash"
echo ""
echo -e "${YELLOW}Tip:${NC} Open two terminal windows, one for each machine!"
echo ""
echo -e "When done, run: ${BOLD}./scripts/demo.sh --cleanup${NC}"
echo -e "Or press Ctrl+C now to cleanup and exit."
echo ""

# Handle --cleanup flag
if [ "${1:-}" = "--cleanup" ]; then
    cleanup
    exit 0
fi

# Drop into Machine A
log "Entering Machine A... (type 'exit' to leave)"
echo ""
trap cleanup EXIT
docker exec -it -e DISPLAY=:99 pipeboard-demo-a bash

exit 0
