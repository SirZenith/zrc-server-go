#! /usr/bin/env python3
import os
from os import read
import re
import sqlite3

import cx_Oracle

source_conn = cx_Oracle.connect('arcaea', 'ARCAEA', 'localhost:1521/xe')
source_cursor = source_conn.cursor()

if os.path.exists('./ArcaeaDB.db'):
    os.remove('./ArcaeaDB.db')
open('./ArcaeaDB.db', 'x').close()

target_conn = sqlite3.connect('./ArcaeaDB.db')
target_cursor = target_conn.cursor()


def create_table():
    with open('./sql_files/build_sqlite.sql', 'r', encoding='utf8') as f:
        content = f.read()
    for stmt in content.split(';'):
        if stmt.isspace():
            continue
        target_cursor.execute(stmt.strip() + ';')


def read_tables():
    tables = [
        'game_info', 'world_map', 'partner',
        'map_reward', 'map_affinity', 'player',
        'friend_list', 'player_map_prog', 'data_backup',
        'pack', 'pack_item', 'pack_purchase_info',
        'song', 'chart_info', 'single',
        'single_purchase_info', 'level_exp', 'part_stats',
        'core', 'core_possess_info', 'world_item',
        'world_unlock', 'world_song', 'world_song_unlock',
        'score', 'best_score', 'recent_score',
        'dl_request'
    ]
    for table in tables:
        source_cursor.execute('select * from %s' % table)
        for result in source_cursor:
            target_cursor.execute(
                'insert into %s values(%s)' % (
                    table, ','.join('?' for i in range(len(result)))
                ), result
            )


if __name__ == '__main__':
    create_table()
    read_tables();
    source_cursor.close()
    source_conn.close()
    target_conn.commit()
    target_cursor.close()
    target_conn.close()
