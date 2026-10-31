-- +goose Up
-- Onomappu editorial text rewritten after reviewing the channel, a second course
-- (natural Japanese) and the video lists of both playlists. Source: the public
-- YouTube channel and playlists, read on 2026-10-23.
-- 000020 is already applied, so each field is replaced only while it still holds
-- the value 000020 wrote: anything an editor changed in the admin wins.
UPDATE class_channels
SET
    -- Custom, not description_youtube: a channel sync overwrites the YouTube value.
    description_custom  = COALESCE(description_custom, 'Onomappu is a YouTube channel by Hitoki, a native Japanese speaker and former medical student in Japan, that teaches Japanese onomatopoeia and natural everyday usage. Most videos are short single-topic lessons explained in easy Japanese with subtitles, often built around personal stories, food and culture.'),
    teaching_style      = CASE WHEN teaching_style = 'Short, funny real-life stories told in slow, clear Japanese, each built around one onomatopoeia and how people actually use it. Subtitles in Japanese, English and Chinese.'
                               THEN 'Short videos (about 4 to 13 minutes) that each focus on one word or usage point. Hitoki explains mostly in easy, natural Japanese with subtitles, using stories and real situations instead of drills.'
                               ELSE teaching_style END,
    pros                = CASE WHEN pros = ARRAY['Teaches onomatopoeia, which textbooks barely cover', 'Native speaker who talks slowly and clearly', 'Subtitles in Japanese, English and Chinese', 'Entertaining stories about real life in Japan']
                               THEN ARRAY['About 40 focused onomatopoeia videos, which is rare on YouTube', 'Clear lessons on words learners often misuse: やばい, じゃん, 大丈夫, は/が, よ/ね, あげる/くれる/もらう', 'Short, self-contained videos that are easy to fit into a daily routine', 'Explanations in natural Japanese double as listening practice']
                               ELSE pros END,
    cons                = CASE WHEN cons = ARRAY['Mostly spoken in Japanese, hard for complete beginners', 'Not a structured grammar course', 'Some videos are culture vlogs rather than lessons']
                               THEN ARRAY['Playlists mix lessons with vlogs, reactions and promotion, so some videos are worth skipping', 'Explanations are mostly in Japanese, so beginners will lean heavily on the subtitles', 'No exercises, quizzes or review built into the playlists', 'A few videos cover mature or flirty topics']
                               ELSE cons END,
    best_for            = CASE WHEN best_for = ARRAY['Learners who know kana and basic grammar', 'Anime and manga fans', 'Learners who want listening practice with subtitles']
                               THEN ARRAY['Late N5 to N3 learners who want to sound more natural', 'Anime and manga fans who keep meeting words like ドキドキ and キラキラ', 'Textbook learners who want to hear how natives really use common words']
                               ELSE best_for END,
    not_for             = CASE WHEN not_for = ARRAY['Absolute beginners who need English explanations', 'Learners preparing for a JLPT exam']
                               THEN ARRAY['Absolute beginners: best once you know kana and basic grammar', 'Learners looking for structured JLPT drills or exam practice', 'Anyone who needs formal or business Japanese']
                               ELSE not_for END,
    verdict             = CASE WHEN verdict = 'The most fun way to learn the sound words Japanese people use every day.'
                               THEN 'A friendly native-speaker supplement for onomatopoeia and casual nuance, not a core course.'
                               ELSE verdict END,
    updated_at          = now()
WHERE slug = 'onomappu';

-- jsonb compares by value, so whitespace in the 000020 literal does not matter.
UPDATE class_courses
SET
    summary            = CASE WHEN summary = 'Learn the sound words (ドキドキ, ゴロゴロ, ギリギリ…) that Japanese people use every day, through short stories in easy Japanese with subtitles.'
                              THEN 'Short native-speaker lessons on everyday onomatopoeia like ドキドキ, イライラ and サクサク, told through stories. Best once you know kana and basic grammar.'
                              ELSE summary END,
    what_you_learn     = CASE WHEN what_you_learn = '["The onomatopoeia every Japanese person uses", "Words for feelings: ドキドキ, もやもや, ワクワク", "Words for food and textures: サクサク, カリカリ, プチプチ", "Everyday situations: ギリギリ, ぴったり, ぎゅうぎゅう", "Listening practice with natural, slow Japanese", "Culture and daily life in Japan"]'
                              THEN '["Use common feeling words like ドキドキ, イライラ and もやもや", "Describe food textures with サクサク, カリカリ, プチプチ and ねばねば", "Tell look-alike pairs apart, such as どんどん and だんだん", "Understand figurative meanings, like カリカリ for an irritable person", "Recognise onomatopoeia you hear in anime and manga", "Follow short explanations given in easy, natural Japanese"]'
                              ELSE what_you_learn END,
    curriculum         = CASE WHEN curriculum = '[{"title": "Start here: words everyone uses", "topics": ["Weird onomatopoeia you actually use", "Onomatopoeia every Japanese person has used", "3 onomatopoeia that help you understand anime", "What ドキドキ really means"]}, {"title": "Feelings and moods", "topics": ["もやもや: life full of unclear feelings", "イチャイチャ", "キレキレ vs キレた", "ボロボロ: worn out", "ワクワク, as Anya says it in Spy x Family"]}, {"title": "Food and textures", "topics": ["プチプチ textures", "Natto onomatopoeia", "サクサク: working like you eat cookies", "カリカリ: crispy snacks", "ヌルヌル: slippery"]}, {"title": "Everyday life in Japan", "topics": ["ギリギリ: only just made it", "ぎゅうぎゅう: packed trains", "ぴったり: arriving right on time", "ノリノリ: surviving a Japanese party", "Convenience store phrases"]}, {"title": "Culture and stories", "topics": ["Learning Japanese at a cat café", "Snack bars for older people", "Dating in the Heian period: こっそり", "A Japanese superstition about lightning", "Breaking school trip rules"]}]'
                              THEN '[
        {"title": "Getting started", "topics": ["Everyday onomatopoeia natives use", "わくわく, パリパリ and つるつる", "ゴロゴロ: lazing around at home", "ギリギリ: just barely", "どんどん vs だんだん"]},
        {"title": "Feelings and moods", "topics": ["ドキドキ beyond romance", "もやもや: a vague, unsettled feeling", "イライラ: feeling irritated", "ビクビク and ブルブル: fear and shivering", "ノリノリ: in high spirits at a party", "ボロボロ: worn out"]},
        {"title": "Food and texture", "topics": ["プチプチ: popping textures", "ねばねば: sticky natto", "サクサク: crunchy, and working smoothly", "カリカリ: crispy, or irritable"]},
        {"title": "One word, many uses", "topics": ["キラキラ: sparkling things and people", "バラバラ: scattered or split up", "ガラガラ: empty, gargling, hoarse", "キレた vs キレキレ", "ぐるぐる: feeling dizzy"]},
        {"title": "People and situations", "topics": ["じろじろ: staring", "ぎゅうぎゅう: a packed train", "ベタベタ: clingy", "ガリガリ: very skinny", "こっそり: doing something secretly", "ぴったり and time etiquette"]},
        {"title": "Anime, culture and stories", "topics": ["Onomatopoeia that help with anime", "Anya''s わくわく in Spy x Family", "モテモテ and モテ期", "ドロドロ: messy relationships", "じゃんけん and its sounds", "Convenience store phrases"]}
    ]'
                              ELSE curriculum END,
    tags               = CASE WHEN tags = ARRAY['Vocabulary', 'Listening', 'Onomatopoeia', 'Culture']
                              THEN ARRAY['Onomatopoeia', 'Vocabulary', 'Listening', 'Culture']
                              ELSE tags END,
    thumbnail_url      = CASE WHEN thumbnail_url = 'https://i.ytimg.com/vi/1XhJo_c-uvA/hqdefault.jpg'
                              THEN 'https://i.ytimg.com/vi/qXoqOvJ9pBE/hqdefault.jpg'
                              ELSE thumbnail_url END,
    total_duration_sec = CASE WHEN total_duration_sec = 0 THEN 24339 ELSE total_duration_sec END,
    updated_at         = now()
WHERE slug = 'onomappu-japanese-onomatopoeia';

INSERT INTO class_courses (
    id, slug, channel_id, title, level, platform, url, playlist_id, thumbnail_url,
    summary, is_free, price_note, what_you_learn, curriculum, tags, total_duration_sec,
    is_published, sort_order
) VALUES (
    'c1a55e00-0000-4000-8000-000000000108', 'onomappu-natural-japanese', (SELECT id FROM class_channels WHERE slug = 'onomappu'),
    'Onomappu — Natural Japanese: Particles, Nuance and Casual Speech', 'N5', 'youtube',
    'https://www.youtube.com/playlist?list=PLNc4a4bdNUdtxwNxBIn6M58BoqQuAqlzr', 'PLNc4a4bdNUdtxwNxBIn6M58BoqQuAqlzr',
    'https://i.ytimg.com/vi/DYQ5YXb-ARU/hqdefault.jpg',
    'A native speaker explains は/が, よ/ね, あげる/くれる/もらう, 大丈夫, じゃん and やばい so you sound natural. Best once you know kana and basic grammar.',
    true, 'Completely free on YouTube',
    '["Choose between は and が, and ある and いる", "Use よ and ね to sound natural, not pushy", "Use あげる, くれる and もらう from the right point of view", "Hear the different meanings of 大丈夫, including a polite no", "Order, apologise and say no politely in real situations", "Use casual じゃん and やばい the way natives do"]',
    '[
        {"title": "Core grammar nuance", "topics": ["ある vs いる", "い-adjectives vs な-adjectives", "あげる, くれる and もらう", "は vs が", "The particle よ", "The particle ね"]},
        {"title": "Everyday situations", "topics": ["Restaurant phrases", "すみません, ごめん and apologising", "The many meanings of 大丈夫", "Saying no without いいえ"]},
        {"title": "Praise and compliments", "topics": ["Replying when told 上手", "Complimenting men and women", "How かわいい is used"]},
        {"title": "Casual speech and slang", "topics": ["Four uses of じゃん", "やばい, good and bad", "Top 5 young people''s slang words", "Can you use anime phrases in real life?", "おじさん構文 texting style", "レロレロ from JoJo"]},
        {"title": "Culture and extra words (optional)", "topics": ["Katakana English (和製英語)", "告白: love-confession culture", "The thunder god and your おへそ", "Superstitions that surprise foreigners", "ぴちぴち and beauty standards"]}
    ]',
    ARRAY['Grammar', 'Particles', 'Nuance', 'Casual speech'], 16650,
    true, 8
) ON CONFLICT DO NOTHING;

-- These rows stand in until the first YouTube sync, which replaces each course's
-- videos as a whole with the same playlist. A course the sync already filled is
-- left alone.
INSERT INTO class_videos (course_id, youtube_video_id, title, thumbnail_url, duration_sec, position)
SELECT c.id, v.youtube_video_id, v.title, 'https://i.ytimg.com/vi/' || v.youtube_video_id || '/hqdefault.jpg', v.duration_sec, v.position
FROM (VALUES
    (1, '1XhJo_c-uvA', 'Weird onomatopoeia that you actually use in Japanese | これが使えればネイティブになれるオノマトペ', 353),
    (2, 'icEnRl5ujDg', 'All the Japanese definitely use these ONOMATOPOEIA | 日本人が絶対使ったことあるオノマトペ | 日本人絕對都用過的擬聲語', 201),
    (3, 'gHNQPG2lLrY', 'アニメがよくわかるようになる3つのオノマトペ｜CC字幕付き(有中文字幕 English subtitles)', 352),
    (4, 'xj8jzM-nwhQ', 'プチプチの食感を説明できる！？', 273),
    (5, 'QFeB4fjwv3o', '日本人はなぜイチャイチャしないのか？', 476),
    (6, 'YetSzTrF2ck', 'コロナウイルスだからゴロゴロしよう | 中文CC字幕、ENG SUB', 319),
    (7, 'yQ3kfjUHujI', 'トイレットペーパーをギリギリ買えた', 293),
    (8, '3fEJxZLSyTo', '竹とんぼ知ってる？どんどんとだんだんは？', 311),
    (9, 'uWAJ6KqwQgM', '日本人の勝負の仕方', 368),
    (10, 'ztXEEmOdr8Q', '「超オススメの日本語学習アプリ」と「オノマトペテスト」！チャンネル登録者1000人🎉', 412),
    (11, 'qXoqOvJ9pBE', 'ドキドキの意味本当に知ってる？', 345),
    (12, '037bGDW7HEw', '人生はもやもやの連続だ', 270),
    (13, 'i1wPMn59-_M', 'やばい人に気をつけて！', 198),
    (14, 'TVfGZKK-uDY', '「あれ」を見た時に使える便利な日本語「cc字幕」', 274),
    (15, 'UCGHL_73_-k', '漢字テストの紙をクシャクシャにしてやる！Frustrated with learning Japanese? KUSHA KUSHA', 449),
    (16, 'gNM6CzlD8lI', '心が傷ついたときに使うオノマトペ｜When your heart STINGS use this JAPANESE (or when you get stung by a giant hornet)', 377),
    (17, 'rtlzIAc6-ss', '画面から納豆を感じるオノマトペ| NATTO | Japanese with subtitles', 622),
    (18, 't8ea1Ah5Y9U', 'よくirritatedになる人はこの日本語覚えよう！| Japanese with Subtitles', 425),
    (19, 'KLfiBjFu9HU', 'キラキラしている未来？Japanese language class', 417),
    (20, 'NdzMoh5cKvg', '道に迷った時に使えるオノマトペ | JUST 1 Japanese onomatopoeia to say ALL of this | バラバラ', 490),
    (21, 'Khc9vsRoTvY', '仏教のオノマトペ？ディズニーで使える！？', 1056),
    (22, 'w4DoF96EgLY', '人生3回しかないモテ期 | THE ONLY 3 TIMES to be POPULAR in your lifetime', 473),
    (23, 'cmdQnXyWQnM', '「日本最大級の猫カフェ」で日本語学ぼう！Cat Cafe Shinjuku Tokyo, Learn Easy Japanese with English Subtitles', 759),
    (24, 'IW3GwdgGv5g', 'シャンプーで転ばないで! Why was the door knob WET? Easy Japanese and some infallible pick up lines | NURU NURU', 660),
    (25, 'fBYgSZi-IEM', 'どのくらいのアルコールで目がグルグルする？ | How much can you DRINK without going GURU GURU? Easy Japanese', 602),
    (26, 'lElPhwPC0OE', 'Curly hair の日本語は何!?', 722),
    (27, 'pg4P0jWoPsk', 'サクサク働くって何？おいしいの？Japanese people WORK the same way they eat COOKIES', 810),
    (28, 'iq1-cJkS7FQ', '日本のお年寄りのバー「スナック」って知ってる？Can you get what they''re saying? Easy Japanese', 527),
    (29, 'rT8EIqf6QYY', 'だらだらした人生を過ごしたい？How not to become GOOD FOR NOTHING - or maybe it''s okay sometimes... | Easy Japanese', 678),
    (30, 'orVLB2PUCTk', 'ベーコンみたいにカリカリしないで！Crispy Japanese Snacks and How Can a Person Be Crispy too?', 628),
    (31, 'hJk2I4iUHz4', 'キレキレ、キレた似ているけど違う意味！ "At that moment I just lost it" Japanese KIRETA | KIRE KIRE | Easy Japanese', 545),
    (32, 'j0ZN_letDrM', 'ニキビができない生活！？ | ぶつぶつ  How not to get PIMPLES? Healthy lifestyle | Easy Japanese', 620),
    (33, 'uNw2AoJf1uk', '時間ぴったりに来ないで！Coming RIGHT ON TIME in Japan is rude? - Japanese rules when it comes to time | PITTARI', 557),
    (34, 'GaqIXHVLulQ', '雷神からおへそを守れ！日本の迷信 A Japanese SUPERSTITION about lightning... and where did PIKACHU''s name come from', 511),
    (35, 'tloKgDgjRGI', 'ハロウィンでの事件　A Halloween Story - how I made a little boy''s Halloween night | Easy Japanese', 572),
    (36, 'rWsUYheOEz8', 'ベタベタされたことある？ A girl from my Japanese cram school did this behind my back ~ easy Japanese storytime', 470),
    (37, 'J-C_Sga17Vk', '電車での悲劇 How come TRAINS in Japan are so PACKED? A childhood storytime | easy Japanese｜ぎゅうぎゅう', 400),
    (38, 'mZ5o0kTXuh0', 'DATING in JAPAN used to be way different in Heian period　あなたは２回デートで結婚できる？ 平安時代 | こっそり', 468),
    (39, 'oKFr17dF2mw', '日本の飲み会で生き残る方法 | Advice for surviving and actually enjoying a Japanese party  | NORI NORI', 679),
    (40, '_uwc63xKIW8', 'Why are Asians so skinny? Med student explains in easy Japanese｜なぜアジア人は痩せやすいのか |  ガリガリ', 443),
    (41, 'wynXnILDZxQ', 'My little sister''s Japanese School Romance Drama もう中高生の恋愛関係は怖いよ ドロドロ', 450),
    (42, 'ol2dnNQl-7c', 'We''re all tired of this - I know I am | BORO BORO | もうみんなボロボロじゃない？', 500),
    (43, 'f_ucP_7FI-E', 'Do we all have a CLONE - can I spot the difference? | もし自分と偽物が入れ替わっていたら？', 324),
    (44, '9Kvh6DJwa3g', 'What are PART TIME JOBS like in JAPAN and how much money do we make?　日本人大学生はどんなバイトしていくらもらえるの？', 398),
    (45, 'oWhsstCIRWU', 'Breaking SCHOOL TRIP rules in Japan - creating embarrassing situations 101  |修学旅行のルールを破る悪ガキ', 407),
    (46, 'srsfM3o2NYM', 'With these COMBINI PHRASES you''ll survive in Japan | Japanese convenience stores', 403),
    (47, 'RGOO1ehnw2M', 'Speak Japanese FLUENTLY after how long?', 491),
    (48, 'ks-fFvyNFYQ', 'Real life DELIQUENTS in my school in Japan - Tokyo Revengers', 520),
    (49, 's-Wr2WcQ1jA', 'Japanese Picture Poses - a guide to impress high school girls in Japan', 511),
    (50, '7vZ3VfdULmc', 'How Anya speaks Japanese wrong - and why it’s cute | Spy x Family WAKU WAKU', 275),
    (51, 'n0zvXwqZD_0', 'How to Know If Someone Likes You', 655)
) AS v(position, youtube_video_id, title, duration_sec)
JOIN class_courses c ON c.slug = 'onomappu-japanese-onomatopoeia'
WHERE NOT EXISTS (SELECT 1 FROM class_videos x WHERE x.course_id = c.id);

INSERT INTO class_videos (course_id, youtube_video_id, title, thumbnail_url, duration_sec, position)
SELECT c.id, v.youtube_video_id, v.title, 'https://i.ytimg.com/vi/' || v.youtube_video_id || '/hqdefault.jpg', v.duration_sec, v.position
FROM (VALUES
    (1, 'XkLQxpD4jp4', 'これで「じゃん」は完璧じゃん！「じゃん」完全マスターYou know right? 4 WAYS to sound like a NATIVE in Japanese using "じゃん"', 837),
    (2, 'VTfa1d0SXHY', '「やばい」の使い方本当にわかってる？', 954),
    (3, 'iq1-cJkS7FQ', '日本のお年寄りのバー「スナック」って知ってる？Can you get what they''re saying? Easy Japanese', 527),
    (4, '4rTOhu6OYI4', '日本の医学部について全部教えます！ Studying Medicine in Japan - Telling You Everything You Want to Know', 701),
    (5, 'nPXdwoo4VZQ', 'なんで日本人はたくさん謝るの！？ Why do Japanese people APOLOGIZE so much? | Easy Japanese', 546),
    (6, 'DmQyBo-H6WU', 'レストランでの日本語全部聞き取れる？Useful PHRASES in Japanese RESTAURANTS - an in depth course | Easy Japanese', 950),
    (7, 'wcPqs5Nabzc', '友達に電話して、日本人がいいえを言わないことを証明します　How do Japanese people say "NO"? | Easy Japanese', 481),
    (8, 'GaqIXHVLulQ', '雷神からおへそを守れ！日本の迷信 A Japanese SUPERSTITION about lightning... and where did PIKACHU''s name come from', 511),
    (9, 'xNH1Z1JUF7c', 'めちゃめちゃ使える日本人の褒め方！How to COMPLIMENT men and women in Japanese | easy Japanese', 832),
    (10, 'VmZwuRsGRX4', '「よ」自然な日本語の助詞の使い方｜ How to sound NATURAL speaking Japanese? PARTICLES: YO', 521),
    (11, 'kE-wapusqLc', '本物のラブレターみたい？日本の告白文化 Japanese LOVE CONFESSION culture - is LOVE LETTER really a thing in Japan?', 501),
    (12, 'FRxge9ZxqxU', '日本で学ぶ外国人医学生！どうやって入学したの？| STUDYING ABROAD in Japan - a Korean MEXT Scholarship Med Student Interview', 1199),
    (13, 'c2SnqEELuIM', 'How do you use "WA" and "GA" in Japanese to sound natural? | Easy Japanese', 481),
    (14, 'EEHfMlZLl1s', 'How to sound NATURAL speaking Japanese? PARTICLES: NE「ね」自然な日本語の助詞の使い方', 536),
    (15, 'EPqJByNllxk', 'My Japanese grandma can speak English even though she can''t - KATAKANA English | おばあちゃんは英語話せないけど、話せる', 408),
    (16, 'yNzC7k03uII', 'Asking Siri out in Japanese - Learning how to use "Daijoubu" with Siri | 大丈夫の使い方をSiriと学ぶ', 435),
    (17, 'DYQ5YXb-ARU', 'Learn to give and receive in Japanese in just 8 minutes - ageru kureru morau「あげる、もらう、くれる」を8分で完璧にする', 525),
    (18, 'pB5O1syozeM', 'How to achieve the East Asian beauty standard? | コラーゲン食べて肌に意味あるのか ｜ぴちぴち', 540),
    (19, 'h39VzDrAMYk', 'Memorizing vocabulary the scientific way 医学的に単語を効率よく覚える方法 Japanese med school student', 421),
    (20, 'OIJrmjN7wh0', 'I called my male friend KAWAII all of a sudden in Japanese | 突然日本の男友達にかわいいって言ってみた', 380),
    (21, '4_5R5cJmTNA', '「ある」と「いる」の使い方絶対間違ってるよ？', 324),
    (22, 'ccu_-O0TbvA', 'TOP 5 Japanese SLANG WORDS on TikTok and Twitter | 若者言葉 top5', 361),
    (23, 'Hosg2MzjR5Y', '日本で行くべき初めてのデートスポット｜First date with a Japanese girl - my favorite spots in Japan', 372),
    (24, 'sc0PAgZoE50', '日本のアニメのフレーズは本当に会話で使えないのか？ - 友達に使って証明する', 574),
    (25, 'bgTLD4XyH4k', 'How Japanese people respond to being told they''re JŌZU - praise in Japan', 506),
    (26, 'h4wBCl0vgTU', 'Why you still can''t speak Japanese (even though you study)', 485),
    (27, 'ouOgFD-vN54', 'Creepy Uncle Slang is Trending in Japan | Ojisan Kōbun', 437),
    (28, 'ZxiSlBa5PUM', 'Japanese Superstitions That Shock Foreigners', 403),
    (29, 'LVMGJ39_fgE', 'Do You REALLY Know How to Use “Rero Rero”? | JoJo', 503),
    (30, 'fYKODK5xI-0', 'How to Never Confuse Japanese Adjectives Again | i- and na-Adjectives', 399)
) AS v(position, youtube_video_id, title, duration_sec)
JOIN class_courses c ON c.slug = 'onomappu-natural-japanese'
WHERE NOT EXISTS (SELECT 1 FROM class_videos x WHERE x.course_id = c.id);

-- +goose Down
-- Editorial text is left in place: an editor may have changed it since.
-- Only the seeded videos go: once a sync has run, the videos are its own.
DELETE FROM class_videos
WHERE course_id = (SELECT id FROM class_courses WHERE slug = 'onomappu-japanese-onomatopoeia' AND last_synced_at IS NULL);
UPDATE class_courses SET total_duration_sec = 0
WHERE slug = 'onomappu-japanese-onomatopoeia' AND last_synced_at IS NULL;
DELETE FROM class_courses WHERE id = 'c1a55e00-0000-4000-8000-000000000108';
