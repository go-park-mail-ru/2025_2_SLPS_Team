Тестирование проводилось на ВМ напрямую к бекенду.
Были получены следующие метики:
Running 3m test @ http://localhost:8080/api/posts
4 threads and 100 connections
Thread Stats   Avg      Stdev     Max   +/- Stdev
Latency   344.54ms  166.89ms   1.83s    76.61%
Req/Sec    74.58     27.40   191.00     73.01%
53337 requests in 3.00m, 73.24MB read
Non-2xx or 3xx responses: 1013
Requests/sec:    296.19
Transfer/sec:    416.46KB
Running 3m test @ http://localhost:8080/api/posts?page=%d&limit=%d0
4 threads and 100 connections
Thread Stats   Avg      Stdev     Max   +/- Stdev
Latency     6.87s     2.04s    9.99s    66.49%
Req/Sec     3.12      5.38    70.00     83.91%
706 requests in 3.00m, 436.66KB read
Socket errors: connect 0, read 0, write 0, timeout 512
Non-2xx or 3xx responses: 81
Requests/sec:      3.92
Transfer/sec:      2.43KB
Также в pgbadger_report.html находится логи postgres преобразованные с помощью pgbadger.

SELECT
p.id,
p.author_id,
p.community_id,
p.text,
p.created_at,
c.name as community_name,
c.avatar_path as community_avatar,
COALESCE(likes.count, 0) AS likes_count,
COALESCE(comments.count, 0) AS comments_count, -- <-- ДОБАВИЛИ
EXISTS (SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = $3) AS liked_by_user
FROM posts p
LEFT JOIN communities c ON p.community_id = c.id
LEFT JOIN (
SELECT post_id, COUNT(*) AS count
FROM post_likes
GROUP BY post_id
) likes ON likes.post_id = p.id
LEFT JOIN ( -- <-- ДОБАВИЛИ
SELECT post_id, COUNT(*) AS count
FROM comments
GROUP BY post_id
) comments ON comments.post_id = p.id
WHERE p.community_id IS NULL
OR p.community_id IN (SELECT community_id FROM community_subscriptions WHERE user_id = $3)
ORDER BY p.created_at DESC
LIMIT $1 OFFSET $2
На чтение выполнялся такой запрос в бд.
Методы оптимизации:
добавить индексы на id, а также покрывающие индексы, 
если это возможно и не отразиться негативно запросах записи.
Денормализовать поля кол-ва комментариев и лайков. 
Разделить посты на новые и старые или по популярности.
Кешировать популярные посты в redis.
Денормализовать поля кол-ва комментариев и лайков или кешировать особенно актуально для популярных постов
потому что кол-во лайков и комментариев будет большим и будет незначительной неточность
Использовать агрегат COUNT(*) FILTER
