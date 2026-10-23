-- +goose Up
-- Banners, playlists and cover images read from the public YouTube pages on
-- 2026-10-23, so the seeded courses look complete before the first API sync.
-- COALESCE keeps anything a sync or an editor has already written.
UPDATE class_channels AS c
SET banner_url_youtube = COALESCE(c.banner_url_youtube, v.banner), updated_at = now()
FROM (VALUES
    ('c1a55e00-0000-4000-8000-000000000001'::uuid, 'https://yt3.googleusercontent.com/MfnWcqAw9go8z47AKbPJsOwt1aJotJMCpUzh0EOfw0N9tNbJE3ckJLLGkpVp1vDMk6cyglE3=w2120-fcrop64=1,00005a57ffffa5a8-k-c0xffffffff-no-nd-rj'),
    ('c1a55e00-0000-4000-8000-000000000002'::uuid, 'https://yt3.googleusercontent.com/N1yd7VVW1cLLJTJMo-QBtEZAKdb-h9kAngHDmCcPPFCRWMjBmfdqtLtMl6-whZMSKDGFJ8FszQ=w2120-fcrop64=1,00005a57ffffa5a8-k-c0xffffffff-no-nd-rj'),
    ('c1a55e00-0000-4000-8000-000000000003'::uuid, 'https://yt3.googleusercontent.com/s4SVUQ_Hpyzyj_aNowD2HE4IBvV2rFWFveFMNWXlWQbBaW3aYZXXpCXVyi-HmCPCJZQvzJlHWQ=w2120-fcrop64=1,00005a57ffffa5a8-k-c0xffffffff-no-nd-rj'),
    ('c1a55e00-0000-4000-8000-000000000004'::uuid, 'https://yt3.googleusercontent.com/5MxpA_nxsQybgEkogiORf4huCNqg0DCRGQLTH20OpdJx6VuF2-WyKJ6Ewmnq0u359SwgeEhvnT0=w2120-fcrop64=1,00005a57ffffa5a8-k-c0xffffffff-no-nd-rj')
) AS v(id, banner)
WHERE c.id = v.id;

UPDATE class_courses AS c
SET playlist_id   = COALESCE(c.playlist_id, v.playlist_id),
    thumbnail_url = COALESCE(c.thumbnail_url, v.thumbnail_url),
    url           = 'https://www.youtube.com/playlist?list=' || COALESCE(c.playlist_id, v.playlist_id),
    updated_at    = now()
FROM (VALUES
    ('c1a55e00-0000-4000-8000-000000000101'::uuid, 'PLd5-Wp_4tLqaDGh1kvlS_N0X3O_bTaKar', 'https://i.ytimg.com/vi/GU1UTcbFwIc/hqdefault.jpg'),
    ('c1a55e00-0000-4000-8000-000000000102'::uuid, 'PLg9uYxuZf8x_A-vcqqyOFZu06WlhnypWj', 'https://i.ytimg.com/vi/pSvH9vH60Ig/hqdefault.jpg'),
    ('c1a55e00-0000-4000-8000-000000000103'::uuid, 'PLA_RcUI8km1NKAoxPzDoIoflnd6bNC7Qm', 'https://i.ytimg.com/vi/GaQBL4XHuSo/hqdefault.jpg'),
    ('c1a55e00-0000-4000-8000-000000000104'::uuid, 'PLINFE8v4DOhtoQSCYikWP_iRyauCeS7mT', 'https://i.ytimg.com/vi/kpR9lZKWWp0/hqdefault.jpg')
) AS v(id, playlist_id, thumbnail_url)
WHERE c.id = v.id;

-- +goose Down
-- Only clears the exact values this migration wrote.
UPDATE class_courses SET playlist_id = NULL, thumbnail_url = NULL,
    url = (SELECT ch.url FROM class_channels ch WHERE ch.id = class_courses.channel_id)
WHERE id IN (
    'c1a55e00-0000-4000-8000-000000000101', 'c1a55e00-0000-4000-8000-000000000102',
    'c1a55e00-0000-4000-8000-000000000103', 'c1a55e00-0000-4000-8000-000000000104'
) AND thumbnail_url LIKE 'https://i.ytimg.com/vi/%/hqdefault.jpg';
UPDATE class_channels SET banner_url_youtube = NULL
WHERE id IN (
    'c1a55e00-0000-4000-8000-000000000001', 'c1a55e00-0000-4000-8000-000000000002',
    'c1a55e00-0000-4000-8000-000000000003', 'c1a55e00-0000-4000-8000-000000000004'
) AND banner_url_youtube LIKE '%=w2120-fcrop64=%';
