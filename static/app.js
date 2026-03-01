// Canoe Slalom Live — Client-side JS

// Auto-refresh leaderboard every 10 seconds
(function() {
    const container = document.getElementById('leaderboard-content');
    if (!container) return;

    const slug = container.dataset.slug;
    let refreshInterval = 10000;
    let paused = false;

    const statusEl = document.getElementById('refresh-status');

    async function refresh() {
        if (paused) return;
        try {
            const resp = await fetch(`/events/${slug}/leaderboard?partial=1`);
            if (resp.ok) {
                const newHTML = await resp.text();
                const oldText = container.innerText;
                container.innerHTML = newHTML;
                const newText = container.innerText;
                if (oldText !== newText) {
                    container.classList.add('leaderboard-updated');
                    setTimeout(() => container.classList.remove('leaderboard-updated'), 1500);
                }
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
