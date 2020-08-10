#! /usr/bin/env python3
import json
import re
from sys import argv

import cx_Oracle

conn = cx_Oracle.connect('arcaea', 'ARCAEA', 'localhost:1521/xe')

cur = conn.cursor()

static_user_id = 1


def player_insertion():
    cur.execute('''insert into player(
            user_id, email, pwdhash, user_name, user_code, ticket, join_date
        )
        values(:1, :2, :3, :4, :5, :6, :7)
    ''', (static_user_id, 'sirzenith@163.com', '222eaf3c32e3665168317399a01af6ef', '陆离', 1, 2000, 1565785560581))
    # cur.execute('''update player set user_name = '陆离' where user_id = 1''')
    cur.execute('select user_name from player')
    for (user_name,) in cur:
        print(user_name)


def level_exp_insertion():
    with open('./json_files/level_exp.json', 'r', encoding='utf8') as f:
        data = json.load(f)
    for lv, exp_val in enumerate(data):
        cur.execute(
            '''insert into level_exp(lv, exp_val) values(:lv, :exp_val)''', lv=lv+1, exp_val=exp_val)

    cur.execute('select lv, exp_val from level_exp')
    for lv, exp_val in cur:
        print('%d: %d' % (lv, exp_val))


def partner_insertion():
    with open('./json_files/partner_info.json', 'r', encoding='utf8') as f:
        partners = json.load(f)
    for partner in partners:
        print(partner['name'])
        cur.execute('''insert into partner(
                part_id, skill_id, skill_id_uncap, char_type, skill_requires_uncap, skill_unlock_level,
                part_name, frag_1, frag_20, prog_1, prog_20, overdrive_1, overdrive_20
            )
            values(
                :1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13
            )''',
                    (
                        partner['character_id'], partner['skill_id'],
                        partner['skill_id_uncap'], partner['char_type'],
                        't' if partner['skill_requires_uncap'] else '',
                        partner['skill_unlock_level'], partner['name'],
                        partner['frag_1'], partner['frag_20'],
                        partner['prog_1'], partner['prog_20'],
                        partner['overdrive_1'], partner['overdrive_20']
                    )
                    )

    cur.execute('select part_id from partner')
    print('Partner IDs: ')
    for part_id in sorted(cur):
        print(part_id[0], end=' ')
    print()


def partner_stats_insertion():
    with open('./json_files/partner_status.json', 'r', encoding='utf8') as f:
        data = json.load(f)
    for status in data:
        try:
            cur.execute('''insert into part_stats(
                    user_id, part_id, is_uncapped_override, is_uncapped, exp_val,
                    overdrive, prog, frag, lv, prog_tempest
                )
                values(:1, :2, :3, :4, :5, :6, :7, :8, :9, :10)''',
                        (
                            static_user_id, status['character_id'],
                            't' if status['is_uncapped_override'] else '',
                            't' if status['is_uncapped'] else '',
                            status['exp'], status['overdrive'], status['prog'],
                            status['frag'], status['level'],
                            60 if status['character_id'] == 35 else 0
                        )
                        )
        except cx_Oracle.IntegrityError as e:
            print(e)
            print(status['level'])
    cur.execute('select part_id from part_stats')
    for (part_id, ) in cur:
        print(part_id, end=' ')
    print()


def pack_info_insertion():
    with open('./json_files/pack_info.json', 'r', encoding='utf8') as f:
        pack_infoes = json.load(f)
    for pack in pack_infoes['packs']:
        pack_info = pack_infoes['detail'][pack]
        cur.execute(
            'insert into pack values(:1, :2, :3, :4, :5)',
            (
                pack, pack_info['price'], pack_info['orig_price'],
                pack_info.get('discount_from', 0), pack_info.get(
                    'discount_to', 0)
            )
        )
        for item in pack_info['items']:
            cur.execute(
                'insert into pack_item values(:1, :2, :3, :4)',
                (
                    pack, item['id'], item['type'],
                    't' if item['is_available'] else ''
                )
            )
        if pack not in ('base', 'single'):
            cur.execute(
                'insert into pack_purchase_info values(:u, :p)',
                u=static_user_id, p=pack
            )


def song_info_insertion():
    with open('./json_files/rating_info.json', 'r', encoding='utf8') as f:
        rating_info = json.load(f)
    with open('../static/songs/songlist', 'r', encoding='utf8') as f:
        song_info = json.load(f)
    with open('../static/songs/checksums.json', 'r', encoding='utf8') as f:
        checksums = json.load(f)

    for song in song_info['songs']:
        song_id = song['id']
        statement = 'insert into song values(%s)' % ', '.join(
            ':' + str(i) for i in range(1, 21))
        print(song_id)
        checksum = checksums[song_id]['audio']['checksum']
        cur.execute(statement, (
            song_id,
            song['title_localized'].get('en', ''),
            song['title_localized'].get('ko', ''),
            song['title_localized'].get('jp', ''),
            song['title_localized'].get('zh-Hant', ''),
            song['title_localized'].get('zh-Hans', ''),
            song['artist'], song['bpm'], song['bpm_base'],
            song['set'], song['purchase'], song['audioPreview'],
            song['audioPreviewEnd'], song['side'],
            't' if song.get('world_unlock', False) else '',
            song['bg'], song['date'], song['version'],
            't' if song.get('remote_dl', False) else '',
            checksum
        ))
        if song['set'] == 'single':
            cur.execute('insert into single values(:song_id)', song_id=song_id)
            cur.execute(
                'insert into single_purchase_info values(:u, :s)',
                u=static_user_id, s=song_id
            )
        if song.get('world_unlock', False):
            cur.execute('insert into world_song values(:item)', item=song_id)
            cur.execute(
                'insert into world_song_unlock values(:1, :2)',
                (static_user_id, song_id)
            )
        for item in song['difficulties']:
            diff = item['ratingClass']
            checksum = checksums[song_id]['chart'][str(diff)]['checksum']
            cur.execute(
                'insert into chart_info values(:1, :2, :3, :4, :5, :6, :7)',
                (
                    song_id, diff,
                    item['chartDesigner'], item.get('jackerDesigner', ''),
                    rating_info[song_id][diff],
                    't' if diff == 3 or song.get('remote_dl', False) else '',
                    checksum
                )
            )
            if diff == 3:
                byd_song_insertion(song_id)


def byd_song_insertion(song_id: str):
    print(song_id)
    song_id += '3'
    cur.execute('insert into world_song(item_name) values(:1)', (song_id,))
    cur.execute(
        'insert into world_song_unlock(user_id, item_name) values(:1, :2)',
        (static_user_id, song_id)
    )


def game_info_insertion():
    cur.execute('insert into game_info values(:1, :2, :3, :4, :5, :6)',
                ('', 12, 1800000, 250, '', 't'))


def map_data_insertion():
    with open('./json_files/map_data.json', 'r', encoding='utf8') as f:
        maps = json.load(f)
    for m in maps:
        print(m['map_id'])
        cur.execute('''insert into world_map(
            available_from, available_to,
            beyond_health,
            chapter, coordinate,
            custom_bg,
            is_beyond, is_legacy, is_repeatable,
            map_id,
            require_id, require_type, require_value,
            stamina_cost, step_count
        ) values(%s)''' % ', '.join(':%d' % i for i in range(1, 16)), (
            m['available_from'], m['available_to'],
            m['beyond_health'],
            m['chapter'], m['coordinate'],
            m['custom_bg'],
            't' if m['is_beyond'] else '',
            't' if m['is_legacy'] else '',
            't' if m['is_repeatable'] else '',
            m['map_id'],
            m['require_id'], m['require_type'], m.get('require_value', None),
            m['stamina_cost'], m['step_count']
        ))
        cur.execute(
            'insert into player_map_prog values(:1, :2, :3, :4, :5)',
            (
                static_user_id, m['map_id'],
                m['curr_capture'], m['curr_position'],
                't' if m['is_locked'] else ''
            )
        )
        for multiplier, character in zip(
                m['affinity_multiplier'], m['character_affinity']):
            cur.execute('insert into map_affinity values(:1, :2, :3)',
                        (m['map_id'], character, multiplier))
        for r in m['rewards']:
            cur.execute(
                'insert into map_reward values(:1, :2, :3, :4, :5)',
                (
                    m['map_id'],
                    r['items'][0].get('id', ''),
                    r['items'][0]['type'],
                    r['items'][0].get('amount', None),
                    r['position']
                )
            )


def world_item_insertion():
    with open('./json_files/world_item.json', 'r', encoding='utf8') as f:
        items = json.load(f)
    for item in items:
        print(item)
        cur.execute('insert into world_item(item_name) values(:1)', (item,))
        cur.execute(
            'insert into world_unlock(user_id, item_name) values(:1, :2)',
            (static_user_id, item)
        )


def core_insertion():
    with open('./json_files/core_info.json', 'r', encoding='utf8') as f:
        cores = json.load(f)
    for core in cores:
        cur.execute(
            'insert into core(core_id, core_name, internal_id) values(:1, :2, :3)',
            (core['id'], core['core_type'], core['_id'])
        )
        cur.execute(
            'insert into core_possess_info(core_id, user_id, amount) values(:1, :2, :3)',
            (core['id'], static_user_id, 0)
        )


def socre_insertion():
    with open('./json_files/scores.json', 'r', encoding='utf8') as f:
        scores = json.load(f)
    scores.sort(key=lambda x: x['rating'], reverse=True)
    for i, score in enumerate(scores):
        print(score['song_id'], score['difficulty'])
        cur.execute(
            '''insert into score values(
            1, :1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13)''',
            (
                int(score['time_played'] / 1000),
                score['song_id'], score['difficulty'], score['score'],
                score['shiny_perfect_count'], score['perfect_count'],
                score['near_count'], score['miss_count'],
                score['rating'], 100, 0, 0, score['clear_type']
            )
        )
        cur.execute('insert into best_score values(1, :1, :2)',
                    (int(score['time_played'] / 1000), score['song_id']))
        if i < 30:
            cur.execute('insert into recent_played values(1, :1)',
                        (int(score['time_played'] / 1000),))
            cur.execute('insert into best_30 values(1, :1)',
                        (int(score['time_played'] / 1000),))
        if i < 10:
            cur.execute('insert into recent_10 values(1, :1)',
                        (int(score['time_played'] / 1000),))


def backup_insertion():
    with open('./json_files/backup_data.json', 'r', encoding='utf8') as f:
        data = f.read().strip('{}')
    data = re.sub('\s', '', data)
    cur.execute(
        'insert into data_backup(user_id, backup_data) values(:1, :2)',
        (static_user_id, data)
    )


if __name__ == '__main__':
    player_insertion()
    level_exp_insertion()
    partner_insertion()
    partner_stats_insertion()
    pack_info_insertion()
    song_info_insertion()
    game_info_insertion()
    map_data_insertion()
    world_item_insertion()
    core_insertion()
    socre_insertion()
    backup_insertion()
    conn.commit()
    cur.close()
    conn.close()
