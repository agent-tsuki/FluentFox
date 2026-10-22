-- +goose Up
-- Ids are fixed so grants and links made against the seed stay valid across
-- environments. Channel facts were checked on 2026-10-22; videos come from sync.
INSERT INTO class_channels (
    id, slug, platform, url, youtube_channel_id, handle,
    name_youtube, avatar_url_youtube, description_youtube, intro_video_id_youtube,
    subscriber_count, accent_color, teaching_style, pros, cons, is_published
) VALUES
(
    'c1a55e00-0000-4000-8000-000000000001', 'japanese-ammo-with-misa', 'youtube',
    'https://www.youtube.com/@JapaneseAmmowithMisa', 'UCBSyd8tXJoEJKIXfrwkPdbA', '@JapaneseAmmowithMisa',
    'Japanese Ammo with Misa',
    'https://yt3.googleusercontent.com/xOk8YBLsc0lES-K1tM0TxOQf_iwgsgtek9rJrxgLei69lxbEk1WKpxtICsQTxDQE3nPzF50zyQ=s240-c-k-c0x00ffffff-no-rj',
    'Misa Sensei (Japanese Ammo with Misa) is a native Japanese teacher based in Japan. She started her YouTube channel to help absolute beginners learn real, conversational Japanese. Her lessons are known for their relaxed pace, clear explanations, and the use of everyday sentences rather than textbook-only phrases.',
    'WYYaf711IEM', 660000, '#E53935',
    'Native Japanese teacher with a warm, story-driven teaching style. Perfect for absolute beginners.',
    ARRAY[
        'Native speaker with exceptional clarity',
        'Extremely beginner-friendly pacing',
        'Natural conversational Japanese (not just textbook)',
        'Free with no account required',
        'Covers all N5 grammar points'
    ],
    ARRAY[
        'No structured JLPT mock tests',
        'Older videos lack consistent formatting',
        'No downloadable PDF notes (community-made PDFs available)'
    ],
    true
),
(
    'c1a55e00-0000-4000-8000-000000000002', 'cure-dolly', 'youtube',
    'https://www.youtube.com/@organicjapanesewithcuredol49', 'UCkdmU8hGK4Fg3LghTVtKltQ', '@organicjapanesewithcuredol49',
    'Organic Japanese with Cure Dolly',
    'https://yt3.googleusercontent.com/ytc/AIdro_nw8mg4x0-3A56gFtGKYZ6QNfQPwTqqmDFuIhpdmOQU8cQ=s240-c-k-c0x00ffffff-no-rj',
    'Cure Dolly created the "Organic Japanese" series based on the philosophy that Japanese is best learned through its own internal logic rather than force-mapped onto English grammatical concepts. The channel, though unconventional in presentation, is widely regarded as one of the most intellectually rigorous approaches to Japanese grammar online.',
    'TkS5APqAFHA', 106000, '#9B59B6',
    'Unconventional structural approach to Japanese grammar. Teaches the language as it actually works, not the Western textbook version.',
    ARRAY[
        'Deepest grammatical explanations available for free',
        'Corrects many misconceptions from textbooks',
        'Builds real intuition for the language',
        'Dense and efficient — no filler content',
        'Great for learners who want to truly understand Japanese'
    ],
    ARRAY[
        'Unconventional presentation style (AI avatar, filtered voice)',
        'Not for complete beginners — some prior kana knowledge needed',
        'No vocabulary or listening practice',
        'Dense videos require rewatching'
    ],
    true
),
(
    'c1a55e00-0000-4000-8000-000000000003', 'tokini-andy', 'youtube',
    'https://www.youtube.com/@ToKiniAndy', 'UCsHzgQRjPzOz_w4HSrNLIVw', '@ToKiniAndy',
    'ToKini Andy',
    'https://yt3.googleusercontent.com/Odm1BnpElzpeAsSWhLXe58XTa_5Fk5h5QtQYwGW4UpjEKqknncffawSNwPIESYhcxYVp6HXLKQ=s240-c-k-c0x00ffffff-no-rj',
    'ToKini Andy follows the popular Genki textbook series chapter by chapter, making the channel an ideal companion resource for university students or self-learners using Genki. Andy''s teaching style is clear, energetic, and exam-aware: he regularly shows how each grammar point maps to JLPT N5/N4 requirements.',
    NULL, 233000, '#E74C3C',
    'Textbook-aligned JLPT prep with Genki I & II. Best for learners who want structured, exam-focused progress.',
    ARRAY[
        'Directly aligned with Genki — most popular Japanese textbook',
        'Explicitly maps to JLPT N5 requirements',
        'Well-structured, chapter-by-chapter progression',
        'Covers kanji systematically',
        'Regular live sessions and Q&A'
    ],
    ARRAY[
        'Best used alongside Genki textbook (extra cost)',
        'Less focus on natural/casual Japanese',
        'Some chapters are long (40-60 min)'
    ],
    true
),
(
    'c1a55e00-0000-4000-8000-000000000004', 'nihongo-no-mori', 'youtube',
    'https://www.youtube.com/@nihongonomori2013', 'UCVx6RFaEAg46xfAsD2zz16w', '@nihongonomori2013',
    '日本語の森 (Nihongo no Mori)',
    'https://yt3.googleusercontent.com/_t0OeOIzQtIRDzEMxlzOTdOSAJhln6BnC3UPFbAadIpXvjmEUWegaJfqB5e75laWE6afeabd=s240-c-k-c0x00ffffff-no-rj',
    'Nihongo no Mori ("Japanese Forest") is a channel run by a team of native Japanese teachers specifically designed for JLPT preparation. Every lesson maps directly to a JLPT exam objective — grammar point, vocabulary category, or kanji reading — in a classroom whiteboard format.',
    '1SGCu28948U', 932000, '#27AE60',
    'JLPT-specific preparation by native Japanese teachers. Laser-focused on passing N5, N4, N3 exams.',
    ARRAY[
        'Most exam-focused free resource available',
        'Native Japanese teachers',
        'Covers all three N5 exam sections (grammar, vocab, kanji)',
        'Professional whiteboard format — easy to follow',
        'Directly usable as structured study plan'
    ],
    ARRAY[
        'Dry, classroom-style — low entertainment value',
        'Assumes some prior Japanese knowledge',
        'Limited cultural context or real-world usage'
    ],
    true
),
(
    'c1a55e00-0000-4000-8000-000000000005', 'japanesepod101', 'website',
    'https://www.japanesepod101.com', NULL, NULL,
    'JapanesePod101', NULL,
    'JapanesePod101 is one of the largest online Japanese learning platforms, operated by Innovative Language. It uses a podcast-style format with two hosts per lesson — one native Japanese speaker and one English-speaking learner — making dialogue feel natural. A free tier gives access to many lessons; the premium plan unlocks learning paths, PDF notes and spaced-repetition vocabulary tools.',
    NULL, NULL, '#2980B9',
    'The most comprehensive structured Japanese course online. Free tier available with 100s of audio/video lessons.',
    ARRAY[
        'Largest library of Japanese lessons online',
        'Audio-first format — great for commuters',
        'Structured learning path by level',
        'Covers cultural context alongside language',
        'Free tier has substantial content'
    ],
    ARRAY[
        'Best features locked behind premium subscription',
        'Can feel sales-y with upgrade prompts',
        'Less grammar depth than dedicated grammar channels',
        'Older lessons have lower production quality'
    ],
    true
),
(
    'c1a55e00-0000-4000-8000-000000000006', 'udemy-miku', 'udemy',
    'https://www.udemy.com/course/japanese-for-beginners-n5/', NULL, NULL,
    'Miku Sensei (Udemy)', NULL,
    'Miku Sensei''s Udemy course is a fully self-paced paid course covering complete N5 preparation with structured modules, quizzes, and downloadable resources. Unlike YouTube channels, the course has a defined beginning and end — ideal for learners who prefer clear milestones and completion certificates.',
    NULL, NULL, '#A435F0',
    'Structured video course on Udemy. Great for learners who prefer a one-time purchase.',
    ARRAY[
        'Full structured course with clear start and end',
        'Includes JLPT mock test — unique among free/cheap options',
        'Downloadable PDF notes for every section',
        'Certificate of completion included',
        'One-time cost — no subscription'
    ],
    ARRAY[
        'Not free (though very cheap during sales)',
        'Less natural/conversational Japanese focus',
        'Cannot ask questions in real-time'
    ],
    true
);

INSERT INTO class_courses (
    id, slug, channel_id, title, level, platform, url, summary, is_free, price_note,
    what_you_learn, curriculum, tags, is_published, sort_order
) VALUES
(
    'c1a55e00-0000-4000-8000-000000000101', 'misa-n5-complete', 'c1a55e00-0000-4000-8000-000000000001',
    'Japanese Ammo — Complete N5 Beginner Course', 'N5', 'youtube',
    'https://www.youtube.com/@JapaneseAmmowithMisa',
    'Native Japanese teacher with a warm, story-driven teaching style. Perfect for absolute beginners.',
    true, 'Completely free on YouTube',
    '["Hiragana & Katakana from scratch", "Essential N5 grammar patterns (は、が、を、に、で、と…)", "Verb conjugation: present, past, te-form, negative", "Adjective types (i-adjectives vs na-adjectives)", "Building natural conversational sentences", "Common daily vocabulary (300+ words)", "Numbers, dates, time expressions", "Polite vs casual speech registers"]',
    '[
        {"title": "Unit 1 — The Writing Systems", "topics": ["Hiragana complete chart", "Katakana complete chart", "Reading practice exercises", "Introduction to Kanji radicals"]},
        {"title": "Unit 2 — Core Grammar Foundations", "topics": ["Sentence structure: Subject + Object + Verb", "Topic particle は vs subject particle が", "Noun + です sentences", "Question formation with か"]},
        {"title": "Unit 3 — Verbs & Conjugation", "topics": ["Verb groups (godan, ichidan, irregular)", "Masu-form (polite present)", "Ta-form (polite past)", "Negative forms (ません、なかった)", "Te-form and its uses"]},
        {"title": "Unit 4 — Adjectives", "topics": ["い-adjectives and conjugation", "な-adjectives and their rules", "Describing nouns", "Adjective + て form for listing"]},
        {"title": "Unit 5 — Essential Particles", "topics": ["Location: に and で", "Direction: へ", "Object: を", "Time: に", "Companion: と", "Tool/means: で"]},
        {"title": "Unit 6 — Everyday Conversation", "topics": ["Greetings and introductions", "Shopping dialogues", "Asking for directions", "Expressing wants: たい", "Can/Cannot: できる"]}
    ]',
    ARRAY['Grammar', 'Vocab', 'Speaking', 'Listening'], true, 1
),
(
    'c1a55e00-0000-4000-8000-000000000102', 'cure-dolly-structural-grammar', 'c1a55e00-0000-4000-8000-000000000002',
    'Organic Japanese with Cure Dolly — Structural Grammar', 'N5', 'youtube',
    'https://www.youtube.com/@organicjapanesewithcuredol49',
    'Unconventional structural approach to Japanese grammar. Teaches the language as it actually works, not the Western textbook version.',
    true, 'Completely free on YouTube',
    '["The true function of は and が (not the Western explanation)", "How Japanese sentence structure actually works", "The logical particle system decoded", "Verb and adjective engines — conjugation as transformation", "How to think in Japanese, not translate from English", "Compound sentences and clause attachment", "The て-form as a connector vs the te-form as a request", "Helper verbs: いる、ある、くれる、もらう、あげる"]',
    '[
        {"title": "Foundations — The Real Structure of Japanese", "topics": ["The Japanese sentence engine", "Null subject and zero pronoun", "は as topic marker (not subject)", "が as the real subject particle"]},
        {"title": "The Logical Particles", "topics": ["を — the object of the verb", "に — the target particle", "で — the contextual particle", "へ、から、まで"]},
        {"title": "Verb Morphology", "topics": ["Verb groups and their stems", "The て-form as a root of many helpers", "ている — ongoing state/action", "てある — resultant state"]},
        {"title": "Adjective System", "topics": ["い-adjectives as verb-like words", "な-adjectives as noun-like words", "Adjectival clauses", "The の nominaliser"]},
        {"title": "Advanced N5 Concepts", "topics": ["Causative: させる", "Passive: られる", "Potential: できる / られる", "Combining clauses with から、ので、が、けど"]}
    ]',
    ARRAY['Grammar', 'Structure', 'Deep Learning', 'Particles'], true, 2
),
(
    'c1a55e00-0000-4000-8000-000000000103', 'tokini-andy-genki-n5', 'c1a55e00-0000-4000-8000-000000000003',
    'Genki I & II Complete Walkthrough — N5/N4 JLPT', 'N5', 'youtube',
    'https://www.youtube.com/@ToKiniAndy',
    'Textbook-aligned JLPT prep with Genki I & II. Best for learners who want structured, exam-focused progress.',
    true, 'Free on YouTube (Genki textbook sold separately)',
    '["All Genki I grammar patterns (N5 range)", "Vocabulary from Genki I chapters 1–12", "JLPT N5 grammar mapping — know exactly what to study", "Reading practice with furigana support", "Listening drills matching Genki audio", "Kanji introduced per chapter (103 N5 kanji)", "Sentence patterns used in JLPT-style questions", "Grammar nuances: differences between similar patterns"]',
    '[
        {"title": "Genki I — Chapters 1–4", "topics": ["Introductions, numbers, time", "これ/それ/あれ、どれ", "X は Y です", "Noun の Noun"]},
        {"title": "Genki I — Chapters 5–8", "topics": ["Verb conjugation (godan & ichidan)", "Direction に/へ, location で", "Past tense verbs and adjectives", "〜たい (want to do)"]},
        {"title": "Genki I — Chapters 9–12", "topics": ["て-form requests and permissions", "いる/いない, ある/ない", "Comparisons: 〜より〜の方が", "Short forms and casual speech"]}
    ]',
    ARRAY['Grammar', 'JLPT Prep', 'Genki', 'Structured'], true, 3
),
(
    'c1a55e00-0000-4000-8000-000000000104', 'nihongo-no-mori-jlpt-n5', 'c1a55e00-0000-4000-8000-000000000004',
    'JLPT N5 Complete Prep — Grammar, Vocab & Kanji', 'N5', 'youtube',
    'https://www.youtube.com/@nihongonomori2013',
    'JLPT-specific preparation by native Japanese teachers. Laser-focused on passing N5, N4, N3 exams.',
    true, 'Completely free on YouTube',
    '["All N5 grammar points in exam-focused order", "N5 vocabulary list (800 words) with usage examples", "All 103 N5 kanji with readings and compounds", "JLPT question format and strategy", "Reading comprehension for short passages", "Listening question types and how to approach them", "Common traps and distractor patterns in N5 questions"]',
    '[
        {"title": "N5 Grammar — All 60+ Points", "topics": ["Sentence enders: ね、よ、か", "Permission and prohibition: ていい、てはいけない", "Ability: ことができる", "Experience: たことがある", "〜ながら (while doing)", "〜てから (after doing)"]},
        {"title": "N5 Vocabulary Drills", "topics": ["Category-based vocab (food, travel, daily life)", "Verb pairs (する/なる, 上げる/上がる)", "Common counters (本、枚、個、匹)", "Time expressions and frequency adverbs"]},
        {"title": "N5 Kanji 103", "topics": ["Kanji by category (numbers, nature, body)", "On-yomi vs kun-yomi drills", "Common compound words", "Writing stroke order guidance"]}
    ]',
    ARRAY['JLPT', 'Grammar', 'Kanji', 'Vocabulary', 'Exam Prep'], true, 4
),
(
    'c1a55e00-0000-4000-8000-000000000105', 'japanesepod101-n5-path', 'c1a55e00-0000-4000-8000-000000000005',
    'JapanesePod101 — Absolute Beginner to N5 Path', 'N5', 'website',
    'https://www.japanesepod101.com',
    'The most comprehensive structured Japanese course online. Free tier available with 100s of audio/video lessons.',
    false, 'Free tier available. Premium from $4/mo',
    '["Survival Japanese for travel and daily life", "Listening comprehension through natural dialogues", "N5 vocabulary in context (not just word lists)", "Grammar explanations with audio examples", "Reading hiragana, katakana, and beginner kanji", "Cultural notes woven into every lesson", "Spaced-repetition vocabulary review (premium)", "Pronunciation and pitch accent basics"]',
    '[
        {"title": "Absolute Beginner Path", "topics": ["Greetings and self-introduction", "Numbers and counting", "Telling time and dates", "Basic shopping dialogue", "Asking questions politely"]},
        {"title": "Beginner Path (N5 Level)", "topics": ["Verb conjugations in context", "Describing people and places", "Making plans and invitations", "Expressing preferences and opinions", "Polite requests and refusals"]},
        {"title": "Vocabulary Builder", "topics": ["1,000 most common Japanese words", "Themed vocab sets (food, work, travel)", "Flashcard-based SRS review", "Word usage in natural sentences"]}
    ]',
    ARRAY['Listening', 'Vocab', 'Grammar', 'Structured Path'], true, 5
),
(
    'c1a55e00-0000-4000-8000-000000000106', 'udemy-miku-complete-n5', 'c1a55e00-0000-4000-8000-000000000006',
    'Complete Japanese N5 — Beginner to JLPT Ready', 'N5', 'udemy',
    'https://www.udemy.com/course/japanese-for-beginners-n5/',
    'Structured video course on Udemy. Often available for under ₹500 during sales. Great for learners who prefer a one-time purchase.',
    false, 'Paid (₹499–₹3,499). Udemy sales frequent.',
    '["Complete N5 grammar syllabus — all 60+ grammar forms", "N5 vocabulary 800 words with audio pronunciation", "All 103 N5 kanji with stroke order and readings", "JLPT N5 full mock test with answer explanations", "Speaking practice — reading aloud exercises", "Section quizzes to test retention at each stage", "Certificate of completion", "Lifetime access + free content updates"]',
    '[
        {"title": "Section 1 — Writing Systems", "topics": ["Hiragana 46 characters", "Katakana 46 characters", "Dakuten, Combination characters", "Introduction to 50 N5 Kanji"]},
        {"title": "Section 2 — Beginner Grammar", "topics": ["Nouns, pronouns, particles", "Adjective types", "Verb groups and basic conjugations", "Question sentences"]},
        {"title": "Section 3 — Intermediate N5 Grammar", "topics": ["て-form and its 8 common uses", "Conditional forms", "Giving/receiving verbs", "Expressing ability, desire, experience"]},
        {"title": "Section 4 — JLPT Mock Test", "topics": ["Full N5 practice test", "Answer walkthrough", "Common mistake analysis", "Score prediction and weak-point review"]}
    ]',
    ARRAY['Grammar', 'Vocab', 'JLPT', 'Certificate', 'Structured'], true, 6
);

-- +goose Down
DELETE FROM class_courses WHERE id IN (
    'c1a55e00-0000-4000-8000-000000000101', 'c1a55e00-0000-4000-8000-000000000102',
    'c1a55e00-0000-4000-8000-000000000103', 'c1a55e00-0000-4000-8000-000000000104',
    'c1a55e00-0000-4000-8000-000000000105', 'c1a55e00-0000-4000-8000-000000000106'
);
DELETE FROM class_channels WHERE id IN (
    'c1a55e00-0000-4000-8000-000000000001', 'c1a55e00-0000-4000-8000-000000000002',
    'c1a55e00-0000-4000-8000-000000000003', 'c1a55e00-0000-4000-8000-000000000004',
    'c1a55e00-0000-4000-8000-000000000005', 'c1a55e00-0000-4000-8000-000000000006'
);
