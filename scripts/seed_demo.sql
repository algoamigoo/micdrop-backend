-- MicDrop interview-demo seed. Safe to re-run (fixed IDs + ON CONFLICT).
-- Run: psql "$DATABASE_URL" -f scripts/seed_demo.sql
-- Demo login user: demo_host (replace google_id with your real one after first OAuth login
-- so the seeded content shows under your account):
--   UPDATE users SET google_id = '<your-google-id>' WHERE user_id = 'demo_host';

-- 1. Users ---------------------------------------------------------------
INSERT INTO users (user_id, user_name, google_id, bio, links, prompt_score, response_score, total_score)
VALUES
  ('demo_host', 'Demo Host', 'demo_google_id_host', 'Comedy night host. Seeding the room.', '"[]"', 0, 0, 0),
  ('comic_ana', 'Comic Ana', 'demo_google_id_ana', 'Observational humor, bad puns.', '"[]"', 0, 0, 0),
  ('joke_joe', 'Joke Joe', 'demo_google_id_joe', 'Dad jokes, professionally.', '"[]"', 0, 0, 0)
ON CONFLICT (user_id) DO NOTHING;

-- 2. Prompts (fixed IDs so re-runs are stable) ---------------------------
INSERT INTO prompts (post_id, user_id, body, prompt_upvotes, response_count)
VALUES
  (1001, 'demo_host', 'My therapist told me to embrace my mistakes... so I hugged my code.', 3, 3),
  (1002, 'demo_host', 'What is the most honest fortune cookie you have ever received?', 2, 2),
  (1003, 'comic_ana', 'I told my plants a joke. Now they are thriving. Coincidence?', 2, 2),
  (1004, 'joke_joe', 'Why do programmers prefer dark mode? Because light attracts bugs.', 1, 1),
  (1005, 'demo_host', 'Drop your best one-liner about Mondays. Winner gets imaginary applause.', 1, 0)
ON CONFLICT (post_id) DO UPDATE SET
  user_id = EXCLUDED.user_id, body = EXCLUDED.body,
  prompt_upvotes = EXCLUDED.prompt_upvotes, response_count = EXCLUDED.response_count;

-- 3. Responses ------------------------------------------------------------
INSERT INTO responses (response_id, post_id, user_id, body, response_upvotes)
VALUES
  (2001, 1001, 'comic_ana', 'Bold move. Did the awkward hug fix anything or just add a sequel?', 2),
  (2002, 1001, 'joke_joe', 'My mistakes filed a restraining order, so I send postcards now.', 2),
  (2003, 1001, 'demo_host', 'Update: the hug helped. The bill did not.', 1),
  (2004, 1002, 'comic_ana', 'You will order takeout again. And like it.', 2),
  (2005, 1002, 'joke_joe', 'Help! I am trapped in a cookie factory.', 1),
  (2006, 1003, 'demo_host', 'My succulent heckled me once. Fair point, honestly.', 2),
  (2007, 1003, 'joke_joe', 'Photosynthesis is just plants laughing in sunlight.', 1),
  (2008, 1004, 'demo_host', 'Explains my 47 browser tabs. All bugs, all attraction.', 1)
ON CONFLICT (response_id) DO UPDATE SET
  post_id = EXCLUDED.post_id, user_id = EXCLUDED.user_id,
  body = EXCLUDED.body, response_upvotes = EXCLUDED.response_upvotes;

-- 4. Prompt votes (self-vote + cross-votes = counters above) --------------
INSERT INTO prompt_votes (user_id, post_id, vote_type) VALUES
  ('demo_host', 1001, 'upvote'), ('comic_ana', 1001, 'upvote'), ('joke_joe', 1001, 'upvote'),
  ('demo_host', 1002, 'upvote'), ('comic_ana', 1002, 'upvote'),
  ('comic_ana', 1003, 'upvote'), ('demo_host', 1003, 'upvote'),
  ('joke_joe', 1004, 'upvote'),
  ('demo_host', 1005, 'upvote')
ON CONFLICT (user_id, post_id) DO UPDATE SET vote_type = EXCLUDED.vote_type;

-- 5. Response votes --------------------------------------------------------
INSERT INTO response_votes (user_id, response_id, vote_type) VALUES
  ('comic_ana', 2001, 'upvote'), ('demo_host', 2001, 'upvote'),
  ('joke_joe', 2002, 'upvote'), ('demo_host', 2002, 'upvote'),
  ('demo_host', 2003, 'upvote'),
  ('comic_ana', 2004, 'upvote'), ('demo_host', 2004, 'upvote'),
  ('joke_joe', 2005, 'upvote'),
  ('demo_host', 2006, 'upvote'), ('comic_ana', 2006, 'upvote'),
  ('joke_joe', 2007, 'upvote'),
  ('demo_host', 2008, 'upvote')
ON CONFLICT (user_id, response_id) DO UPDATE SET vote_type = EXCLUDED.vote_type;

-- 6. Karma (non-self votes only, matches app logic) -----------------------
-- demo_host: prompt +3 (1001 x2, 1002 x1), response +1 (2006) = 4
-- comic_ana: prompt +1 (1003), response +2 (2001, 2004) = 3
-- joke_joe:  prompt +0, response +1 (2002) = 1
UPDATE users SET prompt_score = 3, response_score = 1, total_score = 4 WHERE user_id = 'demo_host';
UPDATE users SET prompt_score = 1, response_score = 2, total_score = 3 WHERE user_id = 'comic_ana';
UPDATE users SET prompt_score = 0, response_score = 1, total_score = 1 WHERE user_id = 'joke_joe';

-- 7. Keep BIGSERIAL sequences ahead of fixed IDs --------------------------
SELECT setval(pg_get_serial_sequence('prompts', 'post_id'), GREATEST((SELECT MAX(post_id) FROM prompts), 1005));
SELECT setval(pg_get_serial_sequence('responses', 'response_id'), GREATEST((SELECT MAX(response_id) FROM responses), 2008));
