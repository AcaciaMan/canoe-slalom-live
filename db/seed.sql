-- Seed data for Demo Slalom 2026
-- Uses INSERT OR IGNORE for idempotency

INSERT OR IGNORE INTO events (id, slug, name, date, location, status, created_at)
VALUES (1, 'demo-slalom-2026', 'Demo Slalom 2026', '2026-06-15', 'Troja Whitewater Course, Prague', 'active', '2026-01-01T00:00:00Z');

-- Categories: K1M and C1W
INSERT OR IGNORE INTO categories (id, event_id, code, name, sort_order, num_runs)
VALUES (1, 1, 'K1M', 'Kayak Single Men', 1, 2);

INSERT OR IGNORE INTO categories (id, event_id, code, name, sort_order, num_runs)
VALUES (2, 1, 'C1W', 'Canoe Single Women', 2, 2);

-- Athletes (10 total)
INSERT OR IGNORE INTO athletes (id, name, club, nation, bio, photo_url, created_at)
VALUES (1, 'Jan Rohan', 'USK Praha', 'CZE',
    'Junior European champion 2024. Known for aggressive lines through upstream gates.',
    '', '2026-01-01T00:00:00Z');

INSERT OR IGNORE INTO athletes (id, name, club, nation, bio, photo_url, created_at)
VALUES (2, 'Oliver Bennett', 'Lee Valley CC', 'GBR',
    'Two-time British senior champion. Excels in technical courses with tight gate spacing.',
    '', '2026-01-01T00:00:00Z');

INSERT OR IGNORE INTO athletes (id, name, club, nation, bio, photo_url, created_at)
VALUES (3, 'Mathieu Deschamps', 'Pau Canoe-Kayak', 'FRA',
    'World Cup bronze medallist 2025. Renowned for his smooth paddling style and clean runs.',
    '', '2026-01-01T00:00:00Z');

INSERT OR IGNORE INTO athletes (id, name, club, nation, bio, photo_url, created_at)
VALUES (4, 'Felix Brauer', 'Augsburg Kanu', 'GER',
    'Former U23 world champion. Strongest in high-water conditions on artificial courses.',
    '', '2026-01-01T00:00:00Z');

INSERT OR IGNORE INTO athletes (id, name, club, nation, bio, photo_url, created_at)
VALUES (5, 'Liam Crawford', 'Penrith Whitewater', 'AUS',
    'Olympic team reserve 2024. Travelled from Sydney specifically for the European circuit.',
    '', '2026-01-01T00:00:00Z');

INSERT OR IGNORE INTO athletes (id, name, club, nation, bio, photo_url, created_at)
VALUES (6, 'Tomáš Kováč', 'Slovak Canoe Club', 'SVK',
    'National champion three years running. Famous for never touching gate 6 in competition.',
    '', '2026-01-01T00:00:00Z');

INSERT OR IGNORE INTO athletes (id, name, club, nation, bio, photo_url, created_at)
VALUES (7, 'Elena Martínez', 'Real Federación Española', 'ESP',
    'European champion 2025 in C1W. Pioneer of the high-brace technique on downstream gates.',
    '', '2026-01-01T00:00:00Z');

INSERT OR IGNORE INTO athletes (id, name, club, nation, bio, photo_url, created_at)
VALUES (8, 'Katarzyna Nowak', 'AZS AWF Kraków', 'POL',
    'Rising star of Polish canoe slalom. Won her first World Cup medal aged 19 in 2025.',
    '', '2026-01-01T00:00:00Z');

INSERT OR IGNORE INTO athletes (id, name, club, nation, bio, photo_url, created_at)
VALUES (9, 'Sophie Leclerc', 'Bourg-Saint-Maurice CK', 'FRA',
    'Consistent top-10 finisher on the World Cup circuit. Known for flawless penalty-free runs.',
    '', '2026-01-01T00:00:00Z');

INSERT OR IGNORE INTO athletes (id, name, club, nation, bio, photo_url, created_at)
VALUES (10, 'Anna Březinová', 'USK Praha', 'CZE',
    'Home favourite at Troja. Junior world silver medallist with a fearless racing style.',
    '', '2026-01-01T00:00:00Z');

-- Entries: 6 in K1M (bibs 101-106), 4 in C1W (bibs 201-204)
INSERT OR IGNORE INTO entries (id, event_id, category_id, athlete_id, bib_number, start_position)
VALUES (1, 1, 1, 1, 101, 1);

INSERT OR IGNORE INTO entries (id, event_id, category_id, athlete_id, bib_number, start_position)
VALUES (2, 1, 1, 2, 102, 2);

INSERT OR IGNORE INTO entries (id, event_id, category_id, athlete_id, bib_number, start_position)
VALUES (3, 1, 1, 3, 103, 3);

INSERT OR IGNORE INTO entries (id, event_id, category_id, athlete_id, bib_number, start_position)
VALUES (4, 1, 1, 4, 104, 4);

INSERT OR IGNORE INTO entries (id, event_id, category_id, athlete_id, bib_number, start_position)
VALUES (5, 1, 1, 5, 105, 5);

INSERT OR IGNORE INTO entries (id, event_id, category_id, athlete_id, bib_number, start_position)
VALUES (6, 1, 1, 6, 106, 6);

INSERT OR IGNORE INTO entries (id, event_id, category_id, athlete_id, bib_number, start_position)
VALUES (7, 1, 2, 7, 201, 1);

INSERT OR IGNORE INTO entries (id, event_id, category_id, athlete_id, bib_number, start_position)
VALUES (8, 1, 2, 8, 202, 2);

INSERT OR IGNORE INTO entries (id, event_id, category_id, athlete_id, bib_number, start_position)
VALUES (9, 1, 2, 9, 203, 3);

INSERT OR IGNORE INTO entries (id, event_id, category_id, athlete_id, bib_number, start_position)
VALUES (10, 1, 2, 10, 204, 4);

-- Seed runs for demo leaderboard
-- K1M runs (entries 1-6)

-- Entry 1 (Jan Rohan, #101): Two runs, Run 1 is better (clean)
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (1, 1, 1, 92450, 0, 0, 0, 92450, 'ok', '2026-06-15T10:05:00Z');
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (2, 1, 2, 89710, 2, 0, 4, 93710, 'ok', '2026-06-15T14:05:00Z');

-- Entry 2 (Oliver Bennett, #102): Two runs, Run 2 is better
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (3, 2, 1, 95330, 1, 0, 2, 97330, 'ok', '2026-06-15T10:08:00Z');
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (4, 2, 2, 91200, 0, 0, 0, 91200, 'ok', '2026-06-15T14:08:00Z');

-- Entry 3 (Mathieu Deschamps, #103): Run 1 has a miss (ouch), Run 2 great recovery
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (5, 3, 1, 88120, 0, 1, 50, 138120, 'ok', '2026-06-15T10:11:00Z');
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (6, 3, 2, 91440, 0, 0, 0, 91440, 'ok', '2026-06-15T14:11:00Z');

-- Entry 4 (Felix Brauer, #104): Only Run 1, with touches
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (7, 4, 1, 93880, 3, 0, 6, 99880, 'ok', '2026-06-15T10:14:00Z');

-- Entry 5 (Liam Crawford, #105): Two clean runs
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (8, 5, 1, 96510, 0, 0, 0, 96510, 'ok', '2026-06-15T10:17:00Z');
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (9, 5, 2, 94230, 0, 0, 0, 94230, 'ok', '2026-06-15T14:17:00Z');

-- Entry 6 (Tomáš Kováč, #106): No runs yet (hasn't started)

-- C1W runs (entries 7-10)

-- Entry 7 (Elena Martínez, #201): Two runs, Run 1 clean and fast
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (10, 7, 1, 98740, 0, 0, 0, 98740, 'ok', '2026-06-15T11:05:00Z');
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (11, 7, 2, 101330, 1, 0, 2, 103330, 'ok', '2026-06-15T15:05:00Z');

-- Entry 8 (Katarzyna Nowak, #202): One run with touches
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (12, 8, 1, 102560, 2, 0, 4, 106560, 'ok', '2026-06-15T11:08:00Z');

-- Entry 9 (Sophie Leclerc, #203): Two clean runs, very consistent
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (13, 9, 1, 100210, 0, 0, 0, 100210, 'ok', '2026-06-15T11:11:00Z');
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (14, 9, 2, 99880, 0, 0, 0, 99880, 'ok', '2026-06-15T15:11:00Z');

-- Entry 10 (Anna Březinová, #204): Run 1 with a miss, Run 2 with touches
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (15, 10, 1, 97650, 0, 1, 50, 147650, 'ok', '2026-06-15T11:14:00Z');
INSERT OR IGNORE INTO runs (id, entry_id, run_number, raw_time_ms, penalty_touches, penalty_misses, penalty_seconds, total_time_ms, status, judged_at)
VALUES (16, 10, 2, 103440, 4, 0, 8, 111440, 'ok', '2026-06-15T15:14:00Z');

-- Sponsors for Demo Slalom 2026
INSERT OR IGNORE INTO sponsors (id, event_id, name, logo_url, website_url, tier, sort_order)
VALUES (1, 1, 'WaterForce Energy', 'https://placehold.co/240x80/1e3a5f/ffffff?text=WaterForce+Energy', 'https://example.com/waterforce', 'main', 1);

INSERT OR IGNORE INTO sponsors (id, event_id, name, logo_url, website_url, tier, sort_order)
VALUES (2, 1, 'PaddleTech', 'https://placehold.co/160x60/2563eb/ffffff?text=PaddleTech', 'https://example.com/paddletech', 'partner', 2);

INSERT OR IGNORE INTO sponsors (id, event_id, name, logo_url, website_url, tier, sort_order)
VALUES (3, 1, 'Alpine Rapids Gear', 'https://placehold.co/160x60/059669/ffffff?text=Alpine+Rapids', 'https://example.com/alpinerapids', 'partner', 3);

INSERT OR IGNORE INTO sponsors (id, event_id, name, logo_url, website_url, tier, sort_order)
VALUES (4, 1, 'River City Tourism', 'https://placehold.co/120x45/6b7280/ffffff?text=River+City', 'https://example.com/rivercity', 'supporter', 4);

INSERT OR IGNORE INTO sponsors (id, event_id, name, logo_url, website_url, tier, sort_order)
VALUES (5, 1, 'CzechPaddle.cz', 'https://placehold.co/120x45/dc2626/ffffff?text=CzechPaddle', 'https://example.com/czechpaddle', 'supporter', 5);

-- Photos for Demo Slalom 2026
-- Athlete action shots
INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (1, 1, 1, 'https://placehold.co/800x600/1e3a5f/ffffff?text=Jan+Rohan+Run+1', 'Jan Rohan navigating gate 12 on Run 1', 'Pavel Novák', '2026-06-15T10:06:00Z');

INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (2, 1, 2, 'https://placehold.co/800x600/2563eb/ffffff?text=Oliver+Bennett+Start', 'Oliver Bennett at the start gate', 'James Wilson', '2026-06-15T10:09:00Z');

INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (3, 1, 7, 'https://placehold.co/800x600/059669/ffffff?text=Elena+Martinez+Finish', 'Elena Martínez crossing the finish line', 'Carlos López', '2026-06-15T11:06:00Z');

INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (4, 1, 3, 'https://placehold.co/800x600/7c3aed/ffffff?text=Mathieu+Deschamps', 'Mathieu Deschamps battling upstream gate 7', 'Pavel Novák', '2026-06-15T10:12:00Z');

INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (5, 1, 10, 'https://placehold.co/800x600/dc2626/ffffff?text=Anna+Brezinova', 'Anna Březinová — home favourite gets the crowd roaring', 'Tereza Dvořáková', '2026-06-15T11:15:00Z');

-- General event photos (no athlete)
INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (6, 1, NULL, 'https://placehold.co/800x600/f59e0b/1a1a1a?text=Troja+Course+Overview', 'Troja Whitewater Course — morning setup', 'Pavel Novák', '2026-06-15T08:30:00Z');

INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (7, 1, NULL, 'https://placehold.co/800x600/10b981/ffffff?text=Finish+Area+Crowd', 'Spectators at the finish area', 'James Wilson', '2026-06-15T12:00:00Z');

INSERT OR IGNORE INTO photos (id, event_id, athlete_id, image_url, caption, photographer_name, created_at)
VALUES (8, 1, NULL, 'https://placehold.co/800x600/6366f1/ffffff?text=Award+Ceremony', 'K1M award ceremony', 'Tereza Dvořáková', '2026-06-15T17:00:00Z');
