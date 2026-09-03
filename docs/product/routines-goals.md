# Routines and Goals

Status: implemented product contract.

## Scope

Trackmate owns two product topics beyond Today and Progress:

- Routines: daily check-ins for repeated habits, with streaks and leaderboard inside the Routines topic.
- Goals: seasonal goals, reviews every two weeks, and final review at the end of the period.

Today remains the daily focus topic: before 20:00 it records one main task,
and after 20:00 without an entry it records one day summary. Progress remains
for closed and auto-failed daily tasks, closed and auto-failed day summaries,
and rare system alerts when Trackmate saved data but Telegram refused to edit
an old message.

## Implemented decisions

1. Add topic keys `routine` and `goals`.
2. Keep `Today` to one daily entry: a main goal-task before 20:00 or a day
   summary after 20:00 when no entry exists for that local date.
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
17. Final seasonal review asks completed, partial, or failed, accepts one or more text messages, and stays open until the author explicitly presses `✅ Завершить итог`.
18. Add rare Today reminders that connect daily tasks with seasonal goals.
19. Clean up ordinary unfinished input after 24 hours silently, deleting the stored bot prompt and known process messages. Two-week and final goal reviews use specialized lifecycles instead.
20. Create the routine card at 08:00 the next day for the previous routine date; remind about an unclosed routine at 20:00 on the check-in day and auto-close missing items as failed at 00:00 the next day while preserving the finalized card and replying to it with the temporary alert.
21. Keep Progress silent: daily result messages do not notify the group.
22. Use pings only for missed/forgotten actions such as routine reminders and missed daily task alerts.
23. Link Progress person labels to the participant profile and link daily result actions/media labels to the source report message.
24. Treat every routine check-in as a snapshot of the routine list that created it. Changing the routine during an active day first snapshots the active day with the old list, then uses the new list for future cards.
25. Keep the user's routine setup message as the source artifact. All-done summaries and auto-close alerts link the word `Рутина`/`рутину` back to that message.
26. In completed routine cards with all items marked, remove the helper line that asks users to mark points; it only belongs to open cards.
27. Keep each unanswered two-week goal review available for 71 hours: replace the first prompt with exactly one identical retry after 24 hours, then remove the retry before Telegram's 48-hour delete limit and persist the review as skipped. If deletion fails, replace the prompt with an inert closed state and continue. Do not overlap the final period review with an open two-week review.
28. Treat stored season boundaries as calendar dates in the workspace timezone. Send the final review on the first local day of the next season.
29. Keep permanent weekly/final cards short and link them to the original Telegram response messages instead of echoing unbounded response text. A permanent final card names its participant using the same `period · person` visual grammar as the seasonal goals card. Create the final prompt as a reply to the stored source goals message when that message still exists, and fall back to a root topic message when Telegram no longer has the reply target.

## Non-goals for MVP

- No complex per-goal parser.
- No per-item goals polling.
- No routine leaderboard in Progress.
- No aggressive Telegram spam.
- No production migration without backup and manual approval.
