// components/KatakanaChart.tsx - Using same CSS as Hiragana
import React, { useState, useEffect } from 'react';
import styles from '../GanaChart.module.css'; // Using the same CSS file

// Structured by rows for transparent table layout - Katakana version
const katakanaData = {
  basic: [
    { row: 'ア行', chars: [
      { character: 'ア', romaji: 'a', example: 'アニメ (anime)' },
      { character: 'イ', romaji: 'i', example: 'イチゴ (strawberry)' },
      { character: 'ウ', romaji: 'u', example: 'ウサギ (rabbit)' },
      { character: 'エ', romaji: 'e', example: 'エレベーター (elevator)' },
      { character: 'オ', romaji: 'o', example: 'オレンジ (orange)' }
    ]},
    { row: 'カ行', chars: [
      { character: 'カ', romaji: 'ka', example: 'カメラ (camera)' },
      { character: 'キ', romaji: 'ki', example: 'キリン (giraffe)' },
      { character: 'ク', romaji: 'ku', example: 'クッキー (cookie)' },
      { character: 'ケ', romaji: 'ke', example: 'ケーキ (cake)' },
      { character: 'コ', romaji: 'ko', example: 'コーヒー (coffee)' }
    ]},
    { row: 'サ行', chars: [
      { character: 'サ', romaji: 'sa', example: 'サッカー (soccer)' },
      { character: 'シ', romaji: 'shi', example: 'シャツ (shirt)' },
      { character: 'ス', romaji: 'su', example: 'スープ (soup)' },
      { character: 'セ', romaji: 'se', example: 'セーター (sweater)' },
      { character: 'ソ', romaji: 'so', example: 'ソファー (sofa)' }
    ]},
    { row: 'タ行', chars: [
      { character: 'タ', romaji: 'ta', example: 'タクシー (taxi)' },
      { character: 'チ', romaji: 'chi', example: 'チーズ (cheese)' },
      { character: 'ツ', romaji: 'tsu', example: 'ツアー (tour)' },
      { character: 'テ', romaji: 'te', example: 'テレビ (TV)' },
      { character: 'ト', romaji: 'to', example: 'トマト (tomato)' }
    ]},
    { row: 'ナ行', chars: [
      { character: 'ナ', romaji: 'na', example: 'ナイフ (knife)' },
      { character: 'ニ', romaji: 'ni', example: 'ニュース (news)' },
      { character: 'ヌ', romaji: 'nu', example: 'ヌードル (noodle)' },
      { character: 'ネ', romaji: 'ne', example: 'ネクタイ (necktie)' },
      { character: 'ノ', romaji: 'no', example: 'ノート (notebook)' }
    ]},
    { row: 'ハ行', chars: [
      { character: 'ハ', romaji: 'ha', example: 'ハンバーガー (hamburger)' },
      { character: 'ヒ', romaji: 'hi', example: 'ヒーロー (hero)' },
      { character: 'フ', romaji: 'fu', example: 'フォーク (fork)' },
      { character: 'ヘ', romaji: 'he', example: 'ヘリコプター (helicopter)' },
      { character: 'ホ', romaji: 'ho', example: 'ホテル (hotel)' }
    ]},
    { row: 'マ行', chars: [
      { character: 'マ', romaji: 'ma', example: 'マウス (mouse)' },
      { character: 'ミ', romaji: 'mi', example: 'ミルク (milk)' },
      { character: 'ム', romaji: 'mu', example: 'ムービー (movie)' },
      { character: 'メ', romaji: 'me', example: 'メニュー (menu)' },
      { character: 'モ', romaji: 'mo', example: 'モニター (monitor)' }
    ]},
    { row: 'ヤ行', chars: [
      { character: 'ヤ', romaji: 'ya', example: 'ヤード (yard)' },
      { character: '', romaji: '', example: '' },
      { character: 'ユ', romaji: 'yu', example: 'ユーザー (user)' },
      { character: '', romaji: '', example: '' },
      { character: 'ヨ', romaji: 'yo', example: 'ヨーグルト (yogurt)' }
    ]},
    { row: 'ラ行', chars: [
      { character: 'ラ', romaji: 'ra', example: 'ライス (rice)' },
      { character: 'リ', romaji: 'ri', example: 'リンゴ (apple)' },
      { character: 'ル', romaji: 'ru', example: 'ルール (rule)' },
      { character: 'レ', romaji: 're', example: 'レストラン (restaurant)' },
      { character: 'ロ', romaji: 'ro', example: 'ロボット (robot)' }
    ]},
    { row: 'ワ行', chars: [
      { character: 'ワ', romaji: 'wa', example: 'ワイン (wine)' },
      { character: '', romaji: '', example: '' },
      { character: '', romaji: '', example: '' },
      { character: '', romaji: '', example: '' },
      { character: 'ヲ', romaji: 'wo', example: 'particle' }
    ]},
    { row: 'ン', chars: [
      { character: 'ン', romaji: 'n', example: 'アンパン (bread)' },
      { character: '', romaji: '', example: '' },
      { character: '', romaji: '', example: '' },
      { character: '', romaji: '', example: '' },
      { character: '', romaji: '', example: '' }
    ]}
  ],
  dakuten: [
    { row: 'ガ行', chars: [
      { character: 'ガ', romaji: 'ga', example: 'ガソリン (gasoline)' },
      { character: 'ギ', romaji: 'gi', example: 'ギター (guitar)' },
      { character: 'グ', romaji: 'gu', example: 'グループ (group)' },
      { character: 'ゲ', romaji: 'ge', example: 'ゲーム (game)' },
      { character: 'ゴ', romaji: 'go', example: 'ゴルフ (golf)' }
    ]},
    { row: 'ザ行', chars: [
      { character: 'ザ', romaji: 'za', example: 'ザッカー (soccer)' },
      { character: 'ジ', romaji: 'ji', example: 'ジュース (juice)' },
      { character: 'ズ', romaji: 'zu', example: 'ズボン (pants)' },
      { character: 'ゼ', romaji: 'ze', example: 'ゼロ (zero)' },
      { character: 'ゾ', romaji: 'zo', example: 'ゾーン (zone)' }
    ]},
    { row: 'ダ行', chars: [
      { character: 'ダ', romaji: 'da', example: 'ダンス (dance)' },
      { character: 'ヂ', romaji: 'di', example: 'ラジオ (radio)' },
      { character: 'ヅ', romaji: 'du', example: 'ヅキ (moon)' },
      { character: 'デ', romaji: 'de', example: 'デスク (desk)' },
      { character: 'ド', romaji: 'do', example: 'ドア (door)' }
    ]},
    { row: 'バ行', chars: [
      { character: 'バ', romaji: 'ba', example: 'バス (bus)' },
      { character: 'ビ', romaji: 'bi', example: 'ビール (beer)' },
      { character: 'ブ', romaji: 'bu', example: 'ブラシ (brush)' },
      { character: 'ベ', romaji: 'be', example: 'ベッド (bed)' },
      { character: 'ボ', romaji: 'bo', example: 'ボール (ball)' }
    ]},
    { row: 'パ行', chars: [
      { character: 'パ', romaji: 'pa', example: 'パン (bread)' },
      { character: 'ピ', romaji: 'pi', example: 'ピザ (pizza)' },
      { character: 'プ', romaji: 'pu', example: 'プール (pool)' },
      { character: 'ペ', romaji: 'pe', example: 'ペン (pen)' },
      { character: 'ポ', romaji: 'po', example: 'ポスト (post)' }
    ]}
  ]
};

interface Props {
  onCharacterClick: (char: any) => void;
  playingAudio: string | null;
}

const KatakanaChart: React.FC<Props> = ({ onCharacterClick, playingAudio }) => {
  const [hoveredChar, setHoveredChar] = useState<string | null>(null);
  const [showRomaji, setShowRomaji] = useState(true);
  const [learnedChars, setLearnedChars] = useState<Set<string>>(new Set());
  const [stickyHeaders, setStickyHeaders] = useState(false);

  const vowelHeaders = ['ア', 'イ', 'ウ', 'エ', 'オ'];

  useEffect(() => {
    const handleScroll = () => {
      setStickyHeaders(window.scrollY > 200);
    };
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  const toggleLearned = (character: string) => {
    const newLearned = new Set(learnedChars);
    if (newLearned.has(character)) {
      newLearned.delete(character);
    } else {
      newLearned.add(character);
    }
    setLearnedChars(newLearned);
  };

  const renderSection = (data: any[], title: string, isDakuten = false) => (
    <div className={styles.tableSection}>
      <div className={styles.sectionHeader}>
        <h3 className={styles.sectionTitle}>
          <span className={styles.titleIcon}>{isDakuten ? 'ガ' : 'カ'}</span>
          {title}
          <div className={styles.charCount}>
            {learnedChars.size} / {isDakuten ? '25' : '46'} learned
          </div>
        </h3>
        
        <div className={styles.controls}>
          <button 
            className={`${styles.controlBtn} ${!showRomaji ? styles.active : ''}`}
            onClick={() => setShowRomaji(!showRomaji)}
          >
            {showRomaji ? 'Hide Romaji' : 'Show Romaji'}
          </button>
        </div>
      </div>

      <div className={styles.transparentTable}>
        {/* Sticky Headers */}
        <div className={`${styles.stickyHeaders} ${stickyHeaders ? styles.visible : ''}`}>
          <div className={styles.cornerHeader}></div>
          {vowelHeaders.map((vowel, index) => (
            <div key={index} className={styles.vowelHeader}>
              {vowel}
            </div>
          ))}
        </div>

        {/* Regular Headers */}
        <div className={styles.tableHeaders}>
          <div className={styles.cornerHeader}></div>
          {vowelHeaders.map((vowel, index) => (
            <div key={index} className={styles.vowelHeader}>
              {vowel}
            </div>
          ))}
        </div>

        {/* Table Body */}
        <div className={styles.tableBody}>
          {data.map((rowData, rowIndex) => (
            <div key={rowIndex} className={styles.tableRow}>
              <div className={styles.rowLabel}>
                {rowData.row}
              </div>
              {rowData.chars.map((char: any, colIndex: number) => (
                <div key={colIndex} className={styles.charCell}>
                  {char.character && (
                    <div 
                      className={`${styles.charCard} ${
                        playingAudio === char.character ? styles.playing : ''
                      } ${learnedChars.has(char.character) ? styles.learned : ''} ${
                        isDakuten ? styles.dakuten : ''
                      }`}
                      onMouseEnter={() => setHoveredChar(char.character)}
                      onMouseLeave={() => setHoveredChar(null)}
                      onClick={() => onCharacterClick(char)}
                      onDoubleClick={() => toggleLearned(char.character)}
                    >
                      <div className={styles.cardContent}>
                        <div className={styles.japaneseChar}>
                          {char.character}
                        </div>
                        <div className={styles.romajiSection}>
                          <div className={`${styles.romajiLabel} ${!showRomaji ? styles.hidden : ''}`}>
                            {char.romaji}
                          </div>
                          {hoveredChar === char.character && char.example && (
                            <div className={styles.exampleTooltip}>
                              {char.example}
                            </div>
                          )}
                        </div>
                      </div>
                      
                      <div className={styles.cardEffects}>
                        <div className={styles.glowEffect}></div>
                        <div className={styles.strokeAnimation}></div>
                      </div>
                      
                      <div className={styles.cardIcons}>
                        <button className={styles.audioIcon} onClick={(e) => {
                          e.stopPropagation();
                          onCharacterClick(char);
                        }}>
                          <svg viewBox="0 0 24 24" fill="currentColor">
                            <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02z"/>
                          </svg>
                        </button>
                        {learnedChars.has(char.character) && (
                          <div className={styles.learnedBadge}>
                            <svg viewBox="0 0 24 24" fill="currentColor">
                              <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                            </svg>
                          </div>
                        )}
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          ))}
        </div>
      </div>
    </div>
  );

  return (
    <div className={styles.hiraganaChart}> {/* Using same class name for consistency */}
      {renderSection(katakanaData.basic, 'Basic Katakana (Gojūon)')}
      {renderSection(katakanaData.dakuten, 'Dakuten (濁点) & Handakuten (半濁点)', true)}
    </div>
  );
};

export default KatakanaChart;
