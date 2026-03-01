// Canoe Slalom Live — Client-side JS

// Auto-refresh leaderboard every 15 seconds
(function() {
    const container = document.getElementById('leaderboard-content');
    if (!container) return;

    const slug = container.dataset.slug;
    let refreshInterval = 15000;
    let paused = false;

    const statusEl = document.getElementById('refresh-status');

    async function refresh() {
        if (paused) return;
        try {
            const resp = await fetch(`/events/${slug}/leaderboard?partial=1`);
            if (resp.ok) {
                container.innerHTML = await resp.text();
                if (statusEl) statusEl.textContent = 'Updated ' + new Date().toLocaleTimeString();
            }
        } catch (e) {
            if (statusEl) statusEl.textContent = 'Refresh failed — retrying...';
        }
    }

    setInterval(refresh, refreshInterval);

    const toggleBtn = document.getElementById('refresh-toggle');
    if (toggleBtn) {
        toggleBtn.addEventListener('click', function() {
            paused = !paused;
            toggleBtn.textContent = paused ? '▶ Resume' : '⏸ Pause';
            if (statusEl) statusEl.textContent = paused ? 'Auto-refresh paused' : 'Auto-refresh active';
        });
    }
})();
