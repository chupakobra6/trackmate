# Routines and Goals

Status: implemented product contract.

## Scope

Trackmate owns two product topics beyond Today and Progress:

- Routines: daily check-ins for repeated habits, with streaks and leaderboard inside the Routines topic.
- Goals: seasonal goals, reviews every two weeks, and final review at the end of the period.

Today remains the daily focus topic. Progress remains for closed daily tasks,
auto-failed daily tasks, and rare system alerts when Trackmate saved data but
Telegram refused to edit an old message.

## Implemented decisions

1. Add topic keys `routine` and `goals`.
2. Keep `Today` as one main goal-task per day.
3. Add a pinned Routines control message with one button: `✏️ Настроить рутину`.
4. Accept routine setup as line-based text, supporting dash/long-dash prefixes and numbered lines.
5. Limit routine items to 9.
6. Treat all routine items as daily in MVP.
7. Send the daily routine check-in at 08:00 the next day for the previous routine date.
8. Use item-by-item routine check-in: done, partial, failed.
9. Ask for a short reason after partial/failed in a separate temporary message, then delete the prompt and the user reply after the answer is saved.
10. Do not ask for a final routine reflection; Today owns the day summary.
11. Publish routine leaderboard only in the Routines topic, not in Progress.
12. Add a Goals control message and seasonal goals format based on result, metric, weekly step, and why.
13. Store goals as raw text in MVP; do not over-parse.
14. Keep Goals setup confirmation concise; do not echo the full goals text as a separate card.
15. Scope unfinished input by topic: Routines, Goals, and Today drafts do not block or cancel each other.
16. Goals review every two weeks asks one combined progress answer.
17. Final seasonal review asks completed, partial, or failed plus a short summary.
18. Add rare Today reminders that connect daily tasks with seasonal goals.
19. Clean up unfinished input after 24 hours silently, deleting the stored bot prompt and known process messages.
20. Create the routine card at 08:00 the next day for the previous routine date; remind about an unclosed routine at 20:00 on the check-in day and auto-close missing items as failed at 00:00 the next day.
21. Keep Progress silent: daily result messages do not notify the group.
22. Use pings only for missed/forgotten actions such as routine reminders and missed daily task alerts.
23. Link Progress person labels to the participant profile and link daily result actions/media labels to the source report message.

## Non-goals for MVP

- No complex per-goal parser.
- No per-item goals polling.
- No routine leaderboard in Progress.
- No aggressive Telegram spam.
- No production migration without backup and manual approval.
