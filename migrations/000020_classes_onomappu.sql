-- +goose Up
-- Onomappu (Hitoki) and his "Onomatopoeia All Videos" playlist. Channel facts,
-- images and the playlist were read from the public YouTube pages on
-- 2026-10-23; the curriculum groups real videos from that playlist by theme.
-- ON CONFLICT DO NOTHING: an environment that already added the channel by
-- hand through the admin keeps its own row, and the course attaches to it by slug.
INSERT INTO class_channels (
    id, slug, platform, url, youtube_channel_id, handle,
    name_youtube, avatar_url_youtube, banner_url_youtube, description_youtube,
    subscriber_count, video_count, accent_color, teaching_style,
    best_for, not_for, pros, cons, verdict, is_published
) VALUES (
    'c1a55e00-0000-4000-8000-000000000007', 'onomappu', 'youtube',
    'https://www.youtube.com/@Onomappu', 'UCLuymDHiOySsAQ9Nc-4NoEQ', '@Onomappu',
    'Onomappu',
    'https://yt3.googleusercontent.com/ud91_Fq3diILbmp6tJF2Y0KUUmbaE9pTq7kINsj6TXZDhaf0Pj8hgvBNIu2cfFlgFNWj_Bv1Zg=s240-c-k-c0x00ffffff-no-rj',
    'https://yt3.googleusercontent.com/s4FFj4czgg44wjJrI9w6pE6bWMdrCUTWyXYoLNNGhcMVGmaNYFzA8EpFOyKxL5i8S1SexH6X=w2120-fcrop64=1,00005a57ffffa5a8-k-c0xffffffff-no-nd-rj',
    'Hitoki is a native Japanese speaker who introduces the onomatopoeia used in everyday Japanese life, with visuals, sounds and short real-life stories. Most videos have Japanese, English and Chinese subtitles.',
    436000, 149, '#00897B',
    'Short, funny real-life stories told in slow, clear Japanese, each built around one onomatopoeia and how people actually use it. Subtitles in Japanese, English and Chinese.',
    ARRAY['Learners who know kana and basic grammar', 'Anime and manga fans', 'Learners who want listening practice with subtitles'],
    ARRAY['Absolute beginners who need English explanations', 'Learners preparing for a JLPT exam'],
    ARRAY[
        'Teaches onomatopoeia, which textbooks barely cover',
        'Native speaker who talks slowly and clearly',
        'Subtitles in Japanese, English and Chinese',
        'Entertaining stories about real life in Japan'
    ],
    ARRAY[
        'Mostly spoken in Japanese, hard for complete beginners',
        'Not a structured grammar course',
        'Some videos are culture vlogs rather than lessons'
    ],
    'The most fun way to learn the sound words Japanese people use every day.',
    true
) ON CONFLICT DO NOTHING;

INSERT INTO class_courses (
    id, slug, channel_id, title, level, platform, url, playlist_id, thumbnail_url,
    summary, is_free, price_note, what_you_learn, curriculum, tags, is_published, sort_order
) VALUES (
    'c1a55e00-0000-4000-8000-000000000107', 'onomappu-japanese-onomatopoeia', (SELECT id FROM class_channels WHERE slug = 'onomappu'),
    'Onomappu — Everyday Japanese Onomatopoeia', 'N5', 'youtube',
    'https://www.youtube.com/playlist?list=PLNc4a4bdNUduYoRLzSZ-IxNENdWT3Fvy_', 'PLNc4a4bdNUduYoRLzSZ-IxNENdWT3Fvy_',
    'https://i.ytimg.com/vi/1XhJo_c-uvA/hqdefault.jpg',
    'Learn the sound words (ドキドキ, ゴロゴロ, ギリギリ…) that Japanese people use every day, through short stories in easy Japanese with subtitles.',
    true, 'Completely free on YouTube',
    '["The onomatopoeia every Japanese person uses", "Words for feelings: ドキドキ, もやもや, ワクワク", "Words for food and textures: サクサク, カリカリ, プチプチ", "Everyday situations: ギリギリ, ぴったり, ぎゅうぎゅう", "Listening practice with natural, slow Japanese", "Culture and daily life in Japan"]',
    '[
        {"title": "Start here: words everyone uses", "topics": ["Weird onomatopoeia you actually use", "Onomatopoeia every Japanese person has used", "3 onomatopoeia that help you understand anime", "What ドキドキ really means"]},
        {"title": "Feelings and moods", "topics": ["もやもや: life full of unclear feelings", "イチャイチャ", "キレキレ vs キレた", "ボロボロ: worn out", "ワクワク, as Anya says it in Spy x Family"]},
        {"title": "Food and textures", "topics": ["プチプチ textures", "Natto onomatopoeia", "サクサク: working like you eat cookies", "カリカリ: crispy snacks", "ヌルヌル: slippery"]},
        {"title": "Everyday life in Japan", "topics": ["ギリギリ: only just made it", "ぎゅうぎゅう: packed trains", "ぴったり: arriving right on time", "ノリノリ: surviving a Japanese party", "Convenience store phrases"]},
        {"title": "Culture and stories", "topics": ["Learning Japanese at a cat café", "Snack bars for older people", "Dating in the Heian period: こっそり", "A Japanese superstition about lightning", "Breaking school trip rules"]}
    ]',
    ARRAY['Vocabulary', 'Listening', 'Onomatopoeia', 'Culture'], true, 7
) ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM class_courses WHERE id = 'c1a55e00-0000-4000-8000-000000000107';
DELETE FROM class_channels WHERE id = 'c1a55e00-0000-4000-8000-000000000007';
