select
        song.song_id,
        song.checksum as "Audio Checksum",
        song.remote_dl as "Song DL",
        to_char(difficulty),
        chart_info.checksum as "Chart Checksum",
        chart_info.remote_dl as "Chart DL"
from
        pack_purchase_info pur, song, chart_info
where
        pur.user_id = 1
        and pur.pack_name = song.pack_name
        and song.song_id = chart_info.song_id
        and (chart_info.remote_dl = 't' or song.remote_dl = 't')
        and song.song_id in ('fairytale')

select * from CHART_INFO where remote_dl = 't';

select * from pack_purchase_info;
