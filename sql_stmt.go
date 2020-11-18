package main

const sqlStmtQueryLoginInfo = `
	select
		user_id, pwdhash from player
	where
		lower(user_name) = lower(?1) or email = ?1
`

const sqlStmtToggleUncap = `
	update part_stats
	set is_uncapped_override =
	case when is_uncapped_override = 't' then
		'f'
	else
		't'
	end
	where part_id = ?
`

const sqlStmtQueryDLInfo = `
	select
		song.song_id,
		song.checksum as "audio_checksum",
		ifnull(song.remote_dl, '') as "song_dl",
		cast(chart_info.difficulty as text) as "difficulty",
		chart_info.checksum as "chart_checksum",
		ifnull(chart_info.remote_dl, '') as "chart_dl"
	from
		%s, song, chart_info
	where
		pur.user_id = ?1
		and song.song_id = chart_info.song_id
		and %s
		and (song.remote_dl = 't' or chart_info.remote_dl = 't')
		%s
`

const sqlStmtOwnedChar = `
	select
		part_id,
		ifnull(is_uncapped_override, '') as uncapped_override,
		ifnull(is_uncapped, '') as uncapped,
		overdrive,
		prog,
		frag,
		prog_tempest,
		part_stats.lv,
		part_stats.exp_val,
		level_exp.exp_val as level_exp
	from
		part_stats, level_exp
	where
		part_stats.user_id = ?1
		and part_stats.lv = level_exp.lv
		`

const sqlStmtCharStaticStats = `
	select 
		ifnull(v.part_id, -1) as has_voice,
		ifnull(skill_id, '') as skill,
		ifnull(skill_id_uncap, '') as skill_uncap,
		ifnull(skill_requires_uncap, '') skill_requires_uncap,
		skill_unlock_level,
		part_name,
		char_type
	from
		partner p left outer join part_voice v on p.part_id = v.part_id 
	where
		p.part_id = ?
`

const sqlStmtSingleCharCond = `and part_stats.part_id = %d`

const sqlStmtChangeChar = `
	update
		player
	set
		partner = ?1, is_skill_sealed = ?2
	where
		user_id = ?3
`

const slqStmtGameInfo = `
	select
		cast(strftime('%s', 'now') as decimal),
		max_stamina,
		stamina_recover_tick,
		core_exp,
		ifnull(world_ranking_enabled, ''),
		ifnull(is_byd_chapter_unlocked, '')
	from
		game_info
`

const sqlStmtLevelStep = `select lv, exp_val from level_exp`

const sqlStmtPackInfo = `
	select
		pack_name, price, orig_price, discount_from, discount_to
	from
		pack
`

const sqlStmtPackItem = `
	select 
		item_id, item_type, is_available
	from
		pack_item
	where
		pack_name = ?
`

const sqlStmtReadBackupData = `select backup_data from data_backup where user_id = ?`

const sqlStmtWriteBackupDate = `update data_backup set backup_data = ?1 where user_id = ?2`

const sqlStmtScoreLookup = `
	select
		sc.played_date,
		so.song_id,
		so.title_local_en,
		sc.difficulty,
		c.rating as base_rating,
		sc.score,
		sc.shiny_pure,
		sc.pure,
		sc.far,
		sc.lost,
		sc.rating,
		sc.health,
		sc.clear_type
	from
		player p, best_score b, score sc, song so, chart_info c
	where
		p.user_code = ?
		and p.user_id = b.user_id
		and p.user_id = sc.user_id
		and sc.played_date = b.played_date
		and so.song_id = sc.song_id
		and so.song_id = c.song_id
		and c.difficulty = sc.difficulty
		and c.rating * 100 > p.rating - 250
	order by
		sc.rating desc
`

const sqlStmtGetScoreLookupRating = `
	with 
		best as (
			select ROW_NUMBER () OVER ( 
				order by rating desc
			) row_num,
			rating
			from  best_score b, score s
			where b.user_id = ?1
				and b.user_id = s.user_id
				and b.played_date = s.played_date
		),
		recent as (
			select rating
			from  recent_score r, score s
			where r.user_id = ?1
				and r.is_recent_10 = 't'
				and r.user_id = s.user_id
				and r.played_date = s.played_date
		)
	select
		ifnull(b30, 0), ifnull(r10, 0)
	from (
		select ifnull(sum(rating), 0) / ifnull(count(rating), 1) b30
		from best
		where row_num <= 30
	), (
		select ifnull(sum(rating), 0) / ifnull(count(rating), 1) r10
		from recent
	)
`

const sqlStmtBaseRating = `
	select rating from chart_info where song_id = ?1 and difficulty = ?2
`

const sqlStmtInsertScore = `
	insert into score (
		user_id,
		played_date,
		song_id,
		difficulty,
		score,
		shiny_pure,
		pure,
		far,
		lost,
		rating,
		health,
		clear_type
	) values(?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)
`

const sqlStmtLookupBestScore = `
	select
		s.score, s.played_date
	from
		best_score b, score s
	where
		b.user_id = ?1
		and b.user_id = s.user_id
		and b.played_date = s.played_date
		and s.song_id = ?2
		and s.difficulty = ?3
`

const sqlStmtInsertBestScore = `
	insert into best_score(user_id, played_date) values(?1, ?2)
`

const sqlStmtReplaceBestScore = `
	update best_score set played_date = ?1 where played_date = ?2
`

const sqlStmtLookupRecentScore = `
	select
		s.rating,
		s.played_date,
		(s.song_id || s.difficulty) iden,
		r.is_recent_10
	from
		recent_score r, score s
	where
		r.user_id = ?1
		and r.user_id = s.user_id
		and r.played_date = s.played_date
`

const sqlStmtReplaceRecnetScore = `
	update recent_score
	set played_date = ?1, is_recent_10 = ?2
	where user_id = ?3 and played_date = ?4
`

const sqlStmtInsertRecentScore = `
	insert into recent_score(user_id, played_date, is_recent_10)
	values(?1, ?2, ?3)
`

const sqlStmtComputeRating = `
	with
		best as (
			select ROW_NUMBER () OVER ( 
				order by rating desc
			) row_num,
			rating
			from  best_score b, score s
			where b.user_id = ?1
				and b.user_id = s.user_id
				and b.played_date = s.played_date
		),
		recent as (
			select rating
			from  recent_score r, score s
			where r.user_id = ?1
				and r.is_recent_10 = 't'
				and r.user_id = s.user_id
				and r.played_date = s.played_date
		)
	select
		round((ifnull(b30, 0) + ifnull(r10, 0)) / (ifnull(b30_count, 1) + ifnull(r10_count, 1)) * 100)
	from (
		select sum(rating) b30, count(rating) b30_count from best
		where row_num <= 30
	), (
		select sum(rating) r10, count(rating) r10_count from recent
	)
`

const sqlStmtUpdateRating = `
	update player set rating = ?1 where user_id = ?2
`

const sqlStmtUserInfo = `
	select
		user_name,
		user_code,
		ifnull(display_name, '') as displayname,
		ticket,
		ifnull(partner, 0) as part_id,
		ifnull(is_locked_name_duplicated, '') as locked,
		ifnull(is_skill_sealed, '') as skill_sealed,
		ifnull(curr_map, '') as curr_map,
		prog_boost,
		stamina,
		next_fragstam_ts,
		max_stamina_ts,
		ifnull(max_stamina_notification, ''),
		ifnull(is_hide_rating, ''), 
		ifnull(favorite_partner, 0),
		recent_score_date, max_friend,
		rating,
		join_date
	from
		player
	where
		user_id = ?
`

const sqlStmtAprilfools = `
	select ifnull(is_aprilfools, '') from game_info
`

const sqlStmtCoreInfo = `
	select
		c.internal_id, c.core_name, amount
	from
		core_possess_info p, core c
	where
		user_id = ?
	and 
		c.core_id = p.core_id
`

const sqlStmtMostRecentScore = `
	select
		s.song_id, s.difficulty, s.score,
		s.shiny_pure, s.pure, s.far, s.lost,
		s.health, s.modifier,
		s.clear_type, s2.clear_type "best clear type"
	from
		score s, best_score b, score s2
	where
		s.user_id = ?1
		and s.played_date = (select max(played_date) from score)
		and s.song_id = s2.song_id
		and b.user_id = ?1
		and b.played_date = s2.played_date
`

const sqlStmtUserSetting = `
	update player set %s = '%s' where user_id = %d
`

const sqlStmtFavouritePartner = `
	update player set favorite_partner = '%d' where user_id = %d
`

const sqlStmtMapInfo = `
	select
		available_from,
		available_to,
		beyond_health,
		chapter,
		coordinate,
		custom_bg,
		ifnull(is_beyond, ''),
		ifnull(is_legacy, ''),
		ifnull(is_repeatable, ''),
		world_map.map_id,
		require_id,
		require_type,
		require_value,
		stamina_cost,
		step_count,
		curr_capture,
		curr_position,
		ifnull(is_locked, '')
	from
		world_map, player_map_prog
	where
		player_map_prog.map_id = world_map.map_id
		and player_map_prog.user_id = ?
`

const sqlStmtCurrentMap = `
	select ifnull(curr_map, '') from player where user_id = ?
`

const sqlStmtMapAffinity = `
	select part_id, multiplier from map_affinity where map_id = ?
`

const sqlStmtRewards = `
	select
		reward_id, item_type, amount, position
	from
		map_reward
	where
		map_id = ?
`
