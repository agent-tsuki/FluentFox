-- +goose Up
-- FoxSensei's own editorial view of each channel. Only empty fields are filled,
-- so anything an editor has already written in the admin wins.
UPDATE class_channels AS c
SET best_for   = CASE WHEN cardinality(c.best_for) = 0 THEN v.best_for ELSE c.best_for END,
    not_for    = CASE WHEN cardinality(c.not_for) = 0 THEN v.not_for ELSE c.not_for END,
    verdict    = CASE WHEN c.verdict = '' THEN v.verdict ELSE c.verdict END,
    updated_at = now()
FROM (VALUES
    ('c1a55e00-0000-4000-8000-000000000001'::uuid,
     ARRAY['Absolute beginners', 'Learners who want natural, spoken Japanese', 'Visual learners who like clear on-screen notes'],
     ARRAY['Learners who want strict JLPT mock tests', 'Anyone looking for a textbook-style syllabus'],
     'The friendliest place to start Japanese grammar from zero.'),
    ('c1a55e00-0000-4000-8000-000000000002'::uuid,
     ARRAY['Learners confused by は vs が', 'Analytical learners who want to know why', 'Self-learners who already read kana'],
     ARRAY['Complete beginners who have not learned kana', 'Learners who want vocabulary or listening practice'],
     'The best free explanation of how Japanese grammar really works.'),
    ('c1a55e00-0000-4000-8000-000000000003'::uuid,
     ARRAY['Genki I & II users', 'Learners preparing for JLPT N5/N4', 'Students who like a fixed chapter order'],
     ARRAY['Learners without the Genki textbook who want a standalone course', 'Anyone focused on casual, spoken Japanese'],
     'The companion to have open next to your Genki textbook.'),
    ('c1a55e00-0000-4000-8000-000000000004'::uuid,
     ARRAY['Learners with a JLPT date booked', 'Learners who already know some Japanese', 'Anyone who likes a classroom format'],
     ARRAY['Learners looking for entertainment value', 'Anyone who wants cultural context and casual speech'],
     'The most direct free route to passing the JLPT.'),
    ('c1a55e00-0000-4000-8000-000000000005'::uuid,
     ARRAY['Commuters who learn by listening', 'Learners who want a guided path by level'],
     ARRAY['Learners who want deep grammar explanations', 'Anyone who dislikes upgrade prompts'],
     'Best for building listening skills on the go.')
) AS v(id, best_for, not_for, verdict)
WHERE c.id = v.id;

-- The seed copied each course summary into teaching_style, so the course page
-- showed the same paragraph twice. Replace only that duplicate.
UPDATE class_channels AS c
SET teaching_style = v.teaching_style, updated_at = now()
FROM (VALUES
    ('c1a55e00-0000-4000-8000-000000000001'::uuid, 'Warm, story-driven explanations with lots of natural example sentences. Each grammar point is broken into small steps and checked with quick practice questions.'),
    ('c1a55e00-0000-4000-8000-000000000002'::uuid, 'Structural and analytical: explains how Japanese sentences are built, then rebuilds each pattern from that core. Dense lessons narrated by an animated avatar.'),
    ('c1a55e00-0000-4000-8000-000000000003'::uuid, 'Clear, energetic, textbook-aligned walkthroughs that show how each grammar point maps to the JLPT N5/N4 requirements.'),
    ('c1a55e00-0000-4000-8000-000000000004'::uuid, 'Classroom whiteboard lessons by native teachers, ordered exactly like the JLPT syllabus. Serious, focused and exam-first.'),
    ('c1a55e00-0000-4000-8000-000000000005'::uuid, 'Audio-first dialogues followed by a breakdown of vocabulary, grammar and cultural notes.')
) AS v(id, teaching_style)
WHERE c.id = v.id
  AND EXISTS (SELECT 1 FROM class_courses co WHERE co.channel_id = c.id AND co.summary = c.teaching_style);

UPDATE class_courses
SET thumbnail_url = 'https://cdn.innovativelanguage.com/sns/thumbnail/site/thumb_fb_japanese.png', updated_at = now()
WHERE id = 'c1a55e00-0000-4000-8000-000000000105' AND thumbnail_url IS NULL;

-- The seeded Udemy course URL does not resolve to a real course (checked
-- 2026-10-23), so hide it until a real course replaces it.
UPDATE class_courses SET is_published = false, updated_at = now() WHERE id = 'c1a55e00-0000-4000-8000-000000000106';
UPDATE class_channels SET is_published = false, updated_at = now() WHERE id = 'c1a55e00-0000-4000-8000-000000000006';

-- +goose Down
-- Editorial text is left in place: an editor may have changed it since.
UPDATE class_channels SET is_published = true WHERE id = 'c1a55e00-0000-4000-8000-000000000006';
UPDATE class_courses SET is_published = true WHERE id = 'c1a55e00-0000-4000-8000-000000000106';
UPDATE class_courses SET thumbnail_url = NULL
WHERE id = 'c1a55e00-0000-4000-8000-000000000105'
  AND thumbnail_url = 'https://cdn.innovativelanguage.com/sns/thumbnail/site/thumb_fb_japanese.png';
