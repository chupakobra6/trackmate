# Trackmate Messages

Редактируемый каталог пользовательских текстов бота.
Каждый блок начинается с `## key`; код импортирует текст по ключу.

## topic.today.title
Сегодня

## topic.routine.title
Рутины

## topic.goals.title
Цели

## topic.progress.title
Прогресс

## season.spring
Весна {{year}}

## season.summer
Лето {{year}}

## season.autumn
Осень {{year}}

## season.winter
Зима {{start_year}}/{{end_year}}

## routine.header_emoji
🌿

## button.setup.check
🔄 Проверить снова

## button.setup.start
✨ Оформить группу

## button.today.add
➕ Добавить задачу

## button.routine.configure
✏️ Настроить рутину

## button.goals.configure
✏️ Настроить цели

## button.dismiss
👀 Понял

## button.routine.done
✅ Да

## button.routine.partial
🔸 Частично

## button.routine.failed
❌ Нет

## button.goal.done
✅ Выполнены

## button.goal.partial
🔸 Частично

## button.goal.failed
❌ Не выполнены

## button.task.report
🏁 Подвести итог

## button.task.done
✅ Выполнена

## button.task.partial
🔸 Выполнена частично

## button.task.failed
❌ Не выполнена

## today.control
🎯 <b>Сегодня</b>
Здесь у каждого одна главная задача дня. Нажми кнопку ниже, чтобы зафиксировать свой главный фокус.

Как это работает:
— Ты формулируешь одну главную задачу дня
— Я закрепляю ее в отдельной карточке
— Вечером в этой же карточке можно отметить итог

## progress.intro
✨ <b>Прогресс</b>
Здесь собирается все важное в аккуратную общую ленту.

Что появляется здесь:
— Выполненные задачи участников
— Автоматические итоги просроченных задач

Так всегда видно, кто что сделал и довел до результата.

## routine.control
🌿 <b>Рутины</b>
Здесь живет одна ежедневная рутина: привычки, задачи и повторяющиеся действия, которые важно держать в ритме.

Нажми кнопку ниже и пришли список. Я буду присылать одну карточку для отметки каждый день после 20:00.

Закрыть ее можно до 12:00 следующего дня.

## goals.control
🎯 <b>Цели</b>
Здесь живут долгосрочные цели, которых мы хотим достичь за сезон, например за лето. Нажми кнопку ниже, чтобы записать свои цели.

Текущий период: <b>Лето 2026</b> (до <b>01.09.2026</b>)

Рекомендую формулировать каждую цель по такой схеме:
1. Направление — сфера жизни или работы, например Спорт, Работа, Языки
— <b>Результат:</b> конкретный и измеримый финал к концу сезона
— <b>Метрика:</b> показатель, по которому будет точно ясно, что цель достигнута
— <b>Шаг недели:</b> регулярное простое действие на каждую неделю
— <b>Зачем:</b> главный смысл цели, почему ее важно держать в фокусе

## setup.ready
✅ <b>Все на месте</b>
Темы и стартовые сообщения в порядке

## setup.repaired
✨ <b>Готово</b>
Пространство оформлено, темы на месте

Что дальше:
— В <b>Сегодня</b> у каждого одна главная задача дня
— В <b>Рутинах</b> — ежедневные отметки и таблица результатов
— В <b>Целях</b> — долгосрочные цели и недельные обзоры
— В <b>Прогрессе</b> — общая лента выполненных задач

## setup.checklist.title
⚙️ <b>Подготовка пространства</b>

## setup.checklist.pending
До запуска нужно закрыть несколько пунктов

## setup.checklist.ready
✅ Можно начинать: все условия выполнены

## setup.checklist.supergroup
Группа переведена в супергруппу

## setup.checklist.forum
Темы включены

## setup.checklist.admin
Бот назначен администратором

## setup.checklist.topics
Бот может управлять темами

## setup.checklist.messages
Бот видит сообщения участников

## setup.checklist.footer
Когда все готово, запускай оформление группы

## daily.card.title
🎯 <b>Задача дня</b> {{person}}

## daily.card.plan
<b>План:</b>

## daily.card.status
<b>Состояние:</b> {{status}}

## daily.card.report
<b>Итог:</b>

## daily.prompt.task
✍️ <b>Напиши главную задачу дня одним сообщением</b>
Можно текстом, голосом, фото или видео

## daily.prompt.report
✍️ <b>Напиши короткий итог одним сообщением</b>
Можно текстом, голосом, фото или видео

## routine.plan.prompt
✍️ <b>Пришли рутину одним сообщением</b>
Это один ежедневный список. Каждый пункт — с новой строки. Удобнее через дефис:

<blockquote>- зарядка
- работа
- английский перед сном
- йога</blockquote>

Нумерацию тоже пойму.

## routine.plan.invalid
⚠️ <b>Пришли список текстом</b>
Каждый пункт должен начинаться с дефиса или номера.

## routine.plan.too_many
⚠️ <b>Слишком много пунктов</b>
Оставь самое важное и пришли список снова.

## routine.card.title
{{emoji}} <b>Рутина за {{date}}</b> {{person}}

## routine.card.subtitle
Отметь пункты за этот день

## routine.item.reason_label
   <i>причина:</i> {{reason}}

## routine.reason.prompt
✍️ <b>Что помешало?</b>

<blockquote>{{item}}</blockquote>

## routine.reminder
🔔 <b>Рутина за {{date}}</b>
Закрой до 12:00

Неотмеченные пункты станут невыполненными

## routine.auto_closed
⚠️ <b>Рутина за {{date}} закрыта</b>
Неотмеченные пункты стали невыполненными

## routine.leaderboard.title
🏆 <b>Таблица рутин</b>

## routine.leaderboard.empty
Пока жду первые завершенные проверки

## routine.leaderboard.best_title
<b>Лучшая серия сезона</b>

## routine.leaderboard.entry
{{rank}}. {{participant}} — {{rate}}% за 7 дней, серия {{streak}} дней, {{items}}

## routine.leaderboard.best_entry
{{participant}} — {{streak}} дней

## routine.items_count.one
{{count}} пункт

## routine.items_count.few
{{count}} пункта

## routine.items_count.many
{{count}} пунктов

## goals.prompt
✍️ <b>Пришли сезонные цели одним сообщением</b>

Текущий период: <b>Лето 2026</b> (до <b>01.09.2026</b>)

Используй для каждой цели эту схему:
<blockquote>1. Направление, например Работа
— Результат: конкретный измеримый итог к концу сезона
— Метрика: как именно ты измеришь успех
— Шаг недели: что делать каждую неделю
— Зачем: почему эта цель важна для тебя</blockquote>

## goals.invalid
⚠️ <b>Пришли цели текстом</b>

## goals.saved
✅ <b>Цели записаны</b>
Недельный обзор придет в воскресенье после 20:00.

## goals.card.title
🎯 <b>Цели на {{period}}</b> · {{person}}

## goals.card.deadline
До <b>{{date}}</b>

## goals.weekly.title
🎯 <b>Недельный обзор целей</b>

## goals.weekly.intro
{{person}}, ответь одним сообщением:

## goals.weekly.q1
1. Что продвинулось по сезонным целям за эту неделю?

## goals.weekly.q2
2. Что мешало двигаться?

## goals.weekly.q3
3. Какой главный шаг берешь на следующую неделю?

## goals.weekly.list_title
<b>Твои цели:</b>

## goals.weekly.saved
✅ <b>Недельный обзор сохранен</b>

## goals.final.title
🏁 <b>Итог периода: {{period}}</b>

## goals.final.ask_status
{{person}}, оцени сезонные цели:

## goals.final.score
<b>Оценка:</b> {{status}}

## goals.final.reflection_intro
Опиши конкретные результаты по целям:

## goals.final.reflection_done
— Что именно удалось довести до конца

## goals.final.reflection_failed
— Что осталось невыполненным и почему

## goals.final.reflection_next
— Какие выводы и задачи переносишь на следующий сезон

## goals.final.saved_summary
<b>Итог:</b>

## progress.custom.default_title
Обновление Trackmate

## progress.system
🔔 Системное сообщение

## progress.daily.task_link
задачу дня

## progress.daily.auto_failed
⏰ <b>{{person}} не выполнил {{task}} вовремя</b>

## progress.daily.closed.done
✅ <b>{{person}} выполнил задачу дня</b>

## progress.daily.closed.partial
🔸 <b>{{person}} частично выполнил задачу дня</b>

## progress.daily.closed.failed
❌ <b>{{person}} не выполнил задачу дня</b>

## progress.daily.closed.default
✅ <b>{{person}} завершил задачу дня</b>

## alert.day_closed_pending_report
🔔 День закончился, а итог по задаче еще не подведен

## alert.auto_failed
⏰ Время вышло. Задача отмечена как не выполненная

## callback.stale_button
Кнопка устарела

## callback.setup.admin_only
Оформить группу может только администратор

## callback.setup.not_ready
Сначала закрой пункты выше, а потом запускай оформление

## callback.workspace_missing
Не получилось найти настройки группы

## callback.notice_hidden
Готово

## callback.alert_hidden
Напоминание скрыто

## callback.today.exists
Задача на сегодня уже зафиксирована

## callback.today.close_previous
Сначала закрой предыдущую задачу

## callback.task.not_found
Задача не найдена

## callback.task.author_only
Итог может оставить только автор задачи

## callback.task.closed
Эта задача уже закрыта

## task.status.prompt
🧾 <b>Выбери итог дня</b>

## task.report.rejected
Итог не принят

## task.report.rejected_closed
Итог не принят: задача уже закрыта

## task.report.saved
✅ <b>Итог сохранен</b>

## routine.checkin.not_found
Проверка не найдена

## routine.checkin.author_only
Отметить рутину может только ее автор

## routine.checkin.completed
Эта проверка уже завершена

## routine.checkin.stale_item
Этот пункт уже не актуален

## goals.not_found
Цели не найдены

## goals.final.author_only
Итог периода может оставить только автор целей

## pending.daily_task_text
Я уже жду формулировку задачи

## pending.daily_task_report
Сначала закончи текущий итог

## pending.routine_plan
Я уже жду рутину

## pending.routine_reason
Я уже жду короткую причину по рутине

## pending.seasonal_goals
Я уже жду сезонные цели

## pending.goal_weekly_review
Сначала закончи недельный обзор целей

## pending.goal_final_reflection
Сначала закончи финальный итог по целям

## pending.default
Сначала закончи текущий ввод

## goal.nudge.failed
Как это событие влияет на твои цели до 1 сентября?

## goal.nudge.done
Как этот итог приближает тебя к сезонным целям?

## goal.nudge.task
Связана ли эта задача с твоими целями на лето?

## daily.status.active
в процессе

## daily.status.awaiting_report
ждет итога

## daily.status.done
выполнена

## daily.status.partial
выполнена частично

## daily.status.failed
не выполнена

## goals.status.done
выполнены

## goals.status.partial
частично

## goals.status.failed
не выполнены

## participant.fallback_name
Без имени

## input.voice
Голосовое сообщение

## input.video_note
Видео-кружок

## input.video
Видео

## input.photo
Фото

## input.audio
Аудио

## input.document
Документ

## input.animation
Анимация

## input.sticker
Стикер

## input.contact
Контакт

## input.location
Локация

## input.venue
Место

## input.poll
Опрос

## input.dice
Кубик

## input.game
Игра

## input.invoice
Счет

## input.message
Сообщение
