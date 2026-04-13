#!/usr/bin/env bash
# Cleans up dev server resources to prevent disk exhaustion.
# Run manually or via cron: 0 3 * * * /path/to/scripts/dev-cleanup.sh >> /var/log/patchiq-cleanup.log 2>&1
set -euo pipefail

echo "=== PatchIQ dev cleanup — $(date) ==="

# Docker: prune stopped containers, dangling images, unused networks, build cache
echo "Docker cleanup..."
docker system prune -f --filter "until=48h" 2>/dev/null || true
docker builder prune -f --filter "until=48h" 2>/dev/null || true

# Go: clean test/build cache older than 7 days
echo "Go cache cleanup..."
go clean -cache 2>/dev/null || true

# Temp build dirs (air hot-reload output)
for user_home in /home/heramb /home/sandy /home/danish /home/rishab; do
    repo="${user_home}/skenzeriq/patchiq"
    if [ -d "${repo}/tmp" ]; then
        find "${repo}/tmp" -type f -mtime +3 -delete 2>/dev/null || true
        echo "Cleaned ${repo}/tmp"
    fi
done

# Git worktree cleanup: remove stale worktrees
for user_home in /home/heramb /home/sandy /home/danish /home/rishab; do
    repo="${user_home}/skenzeriq/patchiq"
    if [ -d "${repo}/.git" ]; then
        (cd "$repo" && git worktree prune 2>/dev/null) || true
    fi
done

# Disk usage report
echo ""
echo "Disk usage:"
df -h / | tail -1
echo ""
echo "Docker usage:"
docker system df 2>/dev/null || true
echo ""
echo "Large files (>100MB) in /home:"
find /home -type f -size +100M -exec ls -lh {} \; 2>/dev/null | head -20 || true

echo ""
echo "=== Cleanup complete ==="
