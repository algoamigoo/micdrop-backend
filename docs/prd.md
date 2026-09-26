1. Overview
MicDrop is a crowdsourced comedy platform inspired by the improv game "Scenes From a Hat" and built on Reddit's community voting model. Users post comedic prompts (setups), others submit one-liner responses (punchlines), and the community upvotes/downvotes to surface the funniest takes. Leaderboards rank users by net karma, and sorting algorithms (Hot, Best, Controversial) help discovery.

Why it works: asynchronous (no need for live interaction), low-friction (one-liners, no essays), and purely merit-driven (community votes decide quality, not recency or clout). Time-decay ranking (Hot sort) keeps fresh content discoverable.
1. User posts a Prompt (e.g. "Things you don't want to hear from your surgeon").
2. Others browse and submit Responses (one-liner punchlines).
3. Community upvotes or downvotes each response; net score ranks them.
4. Leaderboard shows top users by total karma.
5. Sorting (Hot, Top, Best) surfaces the best/newest content; users can sort by preference.

Primary users: comedy writers, comedians, and comedy fans — anyone who wants to riff on prompts, get quick feedback on their jokes, discover funny writers, or just laugh.

Functional Requirements

1. User Management 
    i.unique user_name, a profile showing their prompts, responses and total scores
    ii. prompt_score and response_score (net upvotes)
    iii. user leaderboard based on upvotes

2. Prompts
    i. Any authenticated user can create a Prompt
    ii.Prompts show creation and last-updated timestamps, the author, and all associated responses.
    iii. Prompts can be upvoted by other users.
    iv. Users can browse/list prompts (e.g. newest, most upvoted, most responses).

3. Responses
    i. Any authenticated user can submit a Response (one-liner) to a given prompt.
    ii. A response is tied to exactly one prompt and one author.
    iii. Responses show creation/updated timestamps and an upvote count.
    iv. Responses to a prompt can be sorted (newest, top-voted).

4. Voting
    i. User can upvote or downvote a prompt or response
    ii. one unique vote per prompt or per response by a user
    iii. Voting atomic


Models

User (user_id, user_name, total_score, created_at,updated_at)
Prompts(post_id,user_id,body,prompt_upvotes, created_at,updated_at)
Response (response_id,post_id,user_id,body,reponse_upvotes,created_at,updated_at)

Entity relationships
• users (1) — (many) prompts
• users (1) — (many) responses
• prompts (1) — (many) responses
• users (many) — (many) prompts via prompt_votes
• users (many) — (many) responses via response_votes

