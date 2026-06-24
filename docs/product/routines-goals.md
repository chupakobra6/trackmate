# Routines and Goals

Status: draft for review.

Raw user input is stored in the ChatGPT conversation attachment, not committed here yet.

## Scope

Trackmate should add two product topics:

- Routines: daily check-ins for repeated habits, with streaks and leaderboard inside the Routines topic.
- Goals: seasonal goals, weekly reviews, and final review at the end of the period.

Today remains the daily focus topic. Progress remains only for closed daily tasks and auto-failed daily tasks.

## MVP decisions

1. Add topic keys `routine` and `goals`.
2. Keep `Today` as one main goal-task per day.
3. Add a pinned Routines control message with one button: `✏️ Настроить рутину`.
4. Accept routine setup as line-based text, supporting plain lines, dash bullets, bullet symbols, and numbered lines.
5. Limit routine items to 9.
6. Treat all routine items as daily in MVP.
7. Send the daily routine check-in after 09:00 in the workspace timezone.
8. Use item-by-item routine check-in: done, partial, failed.
9. Ask for a short reason after partial/failed.
10. Ask final reflection: `Что помогло / что помешало / какую одну правку сделаешь завтра?`
11. Publish routine leaderboard only in the Routines topic, not in Progress.
12. Add a Goals control message and seasonal goals format based on result, metric, weekly step, and why.
13. Store goals as raw text in MVP; do not over-parse.
14. Keep Goals setup confirmation concise; do not echo the full goals text as a separate card.
15. Cancel unfinished Routines/Goals setup drafts when the user starts setup in another topic, deleting the previous bot prompt and wrong-topic user message.
16. Weekly goals review asks one combined progress answer.
17. Final seasonal review asks completed, partial, or failed plus a short summary.
18. Add rare Today reminders that connect daily tasks with seasonal goals.

## Non-goals for MVP

- No complex per-goal parser.
- No per-item weekly goals polling.
- No routine leaderboard in Progress.
- No aggressive Telegram spam.
- No production migration without backup and manual approval.
