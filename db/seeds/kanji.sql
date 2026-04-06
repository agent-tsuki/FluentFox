-- db/seeds/kanji.sql
-- Sample N5 kanji entries. Idempotent.

INSERT INTO kanji_entries (character, meaning, on_reading, kun_reading, jlpt_level) VALUES
    ('日', 'sun, day', 'ニチ、ジツ', 'ひ、-び、-か', 'N5'),
    ('月', 'moon, month', 'ゲツ、ガツ', 'つき', 'N5'),
    ('火', 'fire', 'カ', 'ひ', 'N5'),
    ('水', 'water', 'スイ', 'みず', 'N5'),
    ('木', 'tree, wood', 'ボク、モク', 'き、こ', 'N5'),
    ('金', 'gold, money', 'キン、コン', 'かね、かな', 'N5'),
    ('土', 'earth, soil', 'ド、ト', 'つち', 'N5'),
    ('山', 'mountain', 'サン', 'やま', 'N5'),
    ('川', 'river', 'セン', 'かわ', 'N5'),
    ('人', 'person', 'ジン、ニン', 'ひと', 'N5'),
    ('大', 'big', 'ダイ、タイ', 'おお', 'N5'),
    ('小', 'small', 'ショウ', 'ちい、こ、お', 'N5'),
    ('中', 'middle', 'チュウ', 'なか', 'N5'),
    ('国', 'country', 'コク', 'くに', 'N5'),
    ('語', 'language', 'ゴ', 'かた', 'N5')
ON CONFLICT (character) DO NOTHING;
