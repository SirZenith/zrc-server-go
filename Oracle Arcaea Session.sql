select
  sc.played_date,
  so.song_id,
  so.title_local_en,
  sc.difficulty,
  c.rating base_rating sc.score,
  sc.shiny_pure,
  sc.pure,
  sc.far,
  sc.lost,
  sc.rating,
  sc.health,
  sc.clear_type
from player p,
  best_score b,
  score sc,
  song so,
  chart_info c
where
  p.user_code = 1
  and p.user_id = b.user_id
  and p.user_id = sc.user_id
  and sc.played_date = b.played_date
  and sc.rating > (p.rating / 100 - 3)
  and so.song_id = sc.song_id
  and so.song_id = c.song_id
  and c.difficulty = sc.difficulty
order by
  sc.rating desc