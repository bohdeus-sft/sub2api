# PR №1: злиття upstream та перевірка приватності

Дата: 2026-09-23. Форк: bohdeus-sft/sub2api.
База: `7d3e268d9a3bc998bfdde62d270e0928b3cc0cca`.
Upstream PR: `a3eb7ef302961cba716dc78b39b93b60c467db0e` (версія 0.2.8).
PR: https://github.com/bohdeus-sft/sub2api/pull/1

## Результат злиття

PR включає 207 комітів і 360 файлів у diff від спільного предка, а не лише підтримку Sol.
Чотири modify/delete-конфлікти вирішено збереженням видалення:
`deploy/docker-compose.yml`, `docker-compose.local.yml`, `docker-compose.dev.yml`, `docker-compose.standalone.yml`.
Залишено `deploy/docker-compose.personal.yml`: збірка з форку, внутрішні БД/Redis, tmpfs, core=0, ротація Docker-логів та upstream allowlist збережені.
До персонального Compose додано два необов’язкові параметри simple mode з upstream-дефолтами; RUN_MODE не змінено.
Новий Compose CI-тест адаптовано до персональної конфігурації; він також перевіряє відсутність чотирьох видалених файлів.
Незакомічену правку часу використання в AccountsView.vue збережено окремо від злиття.

## Що змінює оновлення

| Область | Зміна та наслідок |
| --- | --- |
| Моделі | GPT-6 Sol/Luna, Claude Opus 5.5, Grok 4.7; каталоги, aliases, тарифи, зображення, reasoning, приклади клієнтських конфігурацій. |
| Sol/Luna | Нормалізація reasoning.mode/effort, збереження none, max, відкидання несумісних sampling-параметрів; tool calls із reasoning потребують Responses. У локальному fallback-каталозі thinking low–max; transport підтримує none окремо. |
| Opus 5.5 | Adaptive thinking, перевірка примусового tool_choice, збереження підписаних thinking-блоків при перетворенні Anthropic ↔ Responses. |
| Streaming | Завершення після terminal event без очікування EOF; виправлення cancel/close HTTP-body, keepalive/ping, TTFT, формату помилок SSE, function_call_arguments.done. |
| Сумісність протоколів | GPT-5.5 Responses Lite після mapping; DeepSeek input_image.url; очищення некоректного tool schema та наддовгих item IDs; ASCII-екранування Codex metadata. |
| Gemini / Antigravity | Вибір thinking-варіантів моделей, змішані каталоги, schema prefixItems, виправлення ідентифікаційного тексту system prompt, транспортний failover та Vertex RetryInfo. |
| Акаунти / планувальник | RPM-параметри в кеші, account-rate fallback, маршрутизація previous_response, збереження налаштувань при OAuth reauth, коректні model_not_found/CF1010 cooldowns і proxy expiry/fallback. |
| Зображення | Сумісні Gemini image API-key маршрути, picker із урахуванням mapping/passthrough, окрема обробка недостатнього балансу. Заблоковані персонально async/batch/storage шляхи не розблоковано. |
| OpenCode Go | Ручне та автоматичне отримання usage-вікон, snapshot для групи одного ключа, debounce/backoff, інвалідація при зміні ключа чи proxy; UI квот. |
| Claude Code | Автоматична синхронізація CLI-версії з GitHub, ручне перевизначення, узгоджені User-Agent/billing headers. |
| Журнали | Налаштовуване rolling retention технічних usage logs; коректне видалення на межі місячної partition; 0 означає відсутність автоматичного очищення, а не заборону запису. |
| Backup | Місячні архіви з окремою політикою retention, захист від неявного видалення, блокування конкурентних restore/cleanup, виправлення повторного шифрування S3 secret. |
| Simple mode | Необов’язкове автостворення default-груп; опційне застосування лімітів витрат API-ключів без списання інших балансів. |
| Фінанси | Ідемпотентна реєстрація offline affiliate withdrawals, виправлення часткових днів і блокувань при списанні підписок, scientific notation та ціна відео за секунду. |
| Інтерфейс | Виправлення stale-response races, IME, dropdown keyboard/focus, date ranges/midnight, scroll lock, скасування image reads, TOTP timers, профілю DingTalk, налаштувань і перекладів. |
| Схема БД | Міграція 240 додає operation_id та унікальний індекс у user_affiliate_ledger. Нових колонок для промптів ця міграція не додає. |
| Залежності | go.mod/go.sum і frontend package/lock-файли цим злиттям не змінено. |

Повний перелік усіх 360 файлів upstream із позначенням відкинутих Compose-файлів: `PR-1-UPSTREAM-FILES.tsv`.

## Висновок щодо вмісту промптів

У переглянутих нових шляхах запису, логування та мережевих викликів не виявлено нового постійного сховища промптів/відповідей або їх надсилання прихованому сторонньому одержувачу. Персональні бар’єри залишилися в коді та пройшли регресійні тести. Це висновок статичного огляду й тестів, а не доказ відсутності будь-яких витоків у всіх конфігураціях.

Збережено:

- `cmd/server` примусово активує privacy policy до ініціалізації.
- `pkg/logger/privacy_core.go` відкидає довільні повідомлення, рядки, помилки, об’єкти, stack traces; `logger.go` блокує DB sink, а log/slog проходять фільтр навіть після reconfigure. Новий slog із upstream_message у ratelimit не обходить цей бар’єр.
- `repository/ops_repo.go`, `audit_log_repo.go`, `content_moderation_repo.go` не записують відповідні журнали при privacy.Enabled().
- `repository/gateway_cache.go` не читає і не пише reasoning content у Redis; `temp_unsched_cache`, scheduler account snapshots і account diagnostics прибирають довільний текст помилок.
- `service/openai_first_output_timeout.go` обмежує буфер RAM і забороняє spill у тимчасовий файл.
- Аудит промптів, зовнішня модерація, плагіни, зовнішня емуляція web search, S3 image storage і асинхронні image tasks залишаються заблокованими.
- Вбудоване оновлення/відкат не може підмінити персональну збірку офіційним бінарником.

## Ремарки та межі

1. **Нове фонове звернення до GitHub увімкнено за замовчуванням.** `claude_code_version_sync_service.go:183` повертає true при відсутньому параметрі й при помилці читання БД. На старті та раз на годину перевіряється release `anthropics/claude-code`. Запит не містить промптів; GitHub бачить мережеву адресу й User-Agent, а за наявності UPDATE_GITHUB_TOKEN — цей GitHub-токен. Для мінімізації зовнішніх звернень вимкнути `claude_code_version_auto_sync_enabled`. Блокування оновлення бінарника цього сервісу не вимикає. Налаштування сервера під час огляду не змінювались.

2. **S3 backup БД не заблокований політикою privacy.** Це існувало раніше; оновлення розширює retention місячними архівами, де 0 означає постійне зберігання. За відсутності налаштувань розклад вимкнений, але існуючі налаштування зберігаються; ручний backup також можливий. Backup містить дані БД, зокрема ключі, налаштування, статистику та старі записи, якщо вони існують. `backup_service.go:603`, `:984`, `backup_retention.go:82`. Це S3 backup БД, окремий від заблокованого S3 image storage. Сам факт злиття не запускає створення backup; стан production-налаштувань не перевірявся. Restore errors можуть залишатись у backup records — не слід вважати цю підсистему універсально очищеною від чутливих даних.

3. **Prompt cache — функція провайдера.** Sol/Luna тепер використовують compat cache key; його hash обчислюється з частин запиту в RAM (`openai_compat_prompt_cache_key.go:30`). Додано передачу `prompt_cache_options`/`prompt_cache_breakpoint` у Responses. Нового локального сховища тексту це не додає, але впливає на кешування провайдера. `store=false` на OAuth/Codex шляху не є гарантією zero retention у провайдера та не є універсальним примусовим правилом для кожного API-key passthrough маршруту.

4. **`encrypted_content` у Claude bridge не завжди ciphertext.** `apicompat/anthropic_to_responses_response.go:20` упаковує thinking, signature/data у Base64 JSON із префіксом `anthropic-thinking-v1:`. Це протокольна обгортка, яку можна декодувати. Вона обробляється в RAM, повертається клієнту та може передаватися назад Anthropic для наступного ходу. Нового запису обгортки на диск/Redis у цьому diff не знайдено. Клієнт може зберігати її у власній історії.

5. **OpenCode usage — додатковий запит із ключем відповідного акаунта.** `opencode_go_usage.go:739`: GET до фіксованого `https://opencode.ai/zen/go/v1/usage`, без тіла промпту, з Bearer API-key; redirects заборонені. Зберігаються typed quota snapshot, timestamps і статичні причини помилок, не raw response body. Глобальний auto-refresh за замовчуванням вимкнений (`:208`), ручний запит доступний адміністратору. OpenAI-only allowlist у Compose не слід трактувати як загальний firewall для всіх фонових HTTP-клієнтів.

6. **Технічна статистика продовжує зберігатися.** Моделі, токени, вартість, час, ідентифікатори, передбачені наявними маршрутами IP/User-Agent та billing/cache hashes — не відсутність будь-яких даних. Новий `request_retention_days=0` вимикає cleanup. Це не спосіб вимкнути журнали. Політика body logging незалежна від цього налаштування.

7. **Старі дані, інфраструктура та provider retention поза цією перевіркою.** Злиття не очищає старі БД/Redis/S3/логи. Coolify/Traefik/CDN/APM, swap/core dumps поза контейнером, історія Codex і налаштування OpenAI перевіряються окремо. Користувацькі значення model/header/metadata також не варто використовувати для передачі секретів, які не потрібні провайдеру.

## Перевірки

- TestPersonalPrivacy: усі 7 пакетів пройшли (server/config/securityaudit/service/repository/middleware/logger), із синтетичними canary-рядками та перевірками заборони записів/мережевої модерації. Початковий запуск securityaudit/repository у sandbox не міг слухати loopback; повтор із дозволеним локальним HTTP пройшов.
- Цільові Go-регресії GPT6/Sol/Luna/Opus55/Backup/OpenCodeGo/ClaudeCodeVersion/RequestLogRetention/RuntimeLog/HTTP body lifecycle: пройшли в 5 пакетах.
- Усі 50 змінених frontend test-файлів: 397 тестів пройшли. Додатково embedded-url privacy та вибрані UI-тести: 37 пройшли.
- Frontend production build і перевірка locale completeness: пройшли; Vite попереджає про великі chunks.
- Backend `go build -tags embed ./cmd/server`: пройшов.
- Compose simple-mode/security checks та git diff whitespace check: пройшли.
- Production БД/міграції, реальні provider-запити і розгортання не виконувались. Повний інтеграційний тест з production-оточенням не заявляється.
