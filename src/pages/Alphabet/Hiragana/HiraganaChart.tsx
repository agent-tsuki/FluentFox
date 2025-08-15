// components/HiraganaChart.tsx - Transparent Table with Modern Cards
import React, { useState, useCallback } from 'react';
import styles from './HiraganaChart.module.css';

// Structured by rows for transparent table layout
const hiraganaData = {
  basic: [
    { row: 'あ行', chars: [
      { character: 'あ', romaji: 'a', example: 'あり (ant)' },
      { character: 'い', romaji: 'i', example: 'いえ (house)' },
      { character: 'う', romaji: 'u', example: 'うみ (sea)' },
      { character: 'え', romaji: 'e', example: 'えき (station)' },
      { character: 'お', romaji: 'o', example: 'おに (demon)' }
    ]},
    { row: 'か行', chars: [
      { character: 'か', romaji: 'ka', example: 'かに (crab)' },
      { character: 'き', romaji: 'ki', example: 'きつね (fox)' },
      { character: 'く', romaji: 'ku', example: 'くも (cloud)' },
      { character: 'け', romaji: 'ke', example: 'けんか (fight)' },
      { character: 'こ', romaji: 'ko', example: 'こども (child)' }
    ]},
    { row: 'さ行', chars: [
      { character: 'さ', romaji: 'sa', example: 'さくら (cherry)' },
      { character: 'し', romaji: 'shi', example: 'しろ (white)' },
      { character: 'す', romaji: 'su', example: 'すし (sushi)' },
      { character: 'せ', romaji: 'se', example: 'せんせい (teacher)' },
      { character: 'そ', romaji: 'so', example: 'そら (sky)' }
    ]},
    { row: 'た行', chars: [
      { character: 'た', romaji: 'ta', example: 'たいよう (sun)' },
      { character: 'ち', romaji: 'chi', example: 'ちいさい (small)' },
      { character: 'つ', romaji: 'tsu', example: 'つき (moon)' },
      { character: 'て', romaji: 'te', example: 'て (hand)' },
      { character: 'と', romaji: 'to', example: 'とり (bird)' }
    ]},
    { row: 'な行', chars: [
      { character: 'な', romaji: 'na', example: 'なまえ (name)' },
      { character: 'に', romaji: 'ni', example: 'にほん (Japan)' },
      { character: 'ぬ', romaji: 'nu', example: 'ぬいぐるみ (doll)' },
      { character: 'ね', romaji: 'ne', example: 'ねこ (cat)' },
      { character: 'の', romaji: 'no', example: 'のど (throat)' }
    ]},
    { row: 'は行', chars: [
      { character: 'は', romaji: 'ha', example: 'はな (flower)' },
      { character: 'ひ', romaji: 'hi', example: 'ひかり (light)' },
      { character: 'ふ', romaji: 'fu', example: 'ふじ (Mt. Fuji)' },
      { character: 'へ', romaji: 'he', example: 'へび (snake)' },
      { character: 'ほ', romaji: 'ho', example: 'ほん (book)' }
    ]},
    { row: 'ま行', chars: [
      { character: 'ま', romaji: 'ma', example: 'まつり (festival)' },
      { character: 'み', romaji: 'mi', example: 'みず (water)' },
      { character: 'む', romaji: 'mu', example: 'むし (bug)' },
      { character: 'め', romaji: 'me', example: 'め (eye)' },
      { character: 'も', romaji: 'mo', example: 'もり (forest)' }
    ]},
    { row: 'や行', chars: [
      { character: 'や', romaji: 'ya', example: 'やま (mountain)' },
      { character: '', romaji: '', example: '' },
      { character: 'ゆ', romaji: 'yu', example: 'ゆき (snow)' },
      { character: '', romaji: '', example: '' },
      { character: 'よ', romaji: 'yo', example: 'よる (night)' }
    ]},
    { row: 'ら行', chars: [
      { character: 'ら', romaji: 'ra', example: 'らいおん (lion)' },
      { character: 'り', romaji: 'ri', example: 'りんご (apple)' },
      { character: 'る', romaji: 'ru', example: 'るーる (rule)' },
      { character: 'れ', romaji: 're', example: 'れんしゅう (practice)' },
      { character: 'ろ', romaji: 'ro', example: 'ろうそく (candle)' }
    ]},
    { row: 'わ行', chars: [
      { character: 'わ', romaji: 'wa', example: 'わたし (I/me)' },
      { character: '', romaji: '', example: '' },
      { character: '', romaji: '', example: '' },
      { character: '', romaji: '', example: '' },
      { character: 'を', romaji: 'wo', example: 'particle' }
    ]},
    { row: 'ん', chars: [
      { character: 'ん', romaji: 'n', example: 'ほん (book)' },
      { character: '', romaji: '', example: '' },
      { character: '', romaji: '', example: '' },
      { character: '', romaji: '', example: '' },
      { character: '', romaji: '', example: '' }
    ]}
  ],
  dakuten: [
    { row: 'が行', chars: [
      { character: 'が', romaji: 'ga', example: 'がっこう (school)' },
      { character: 'ぎ', romaji: 'gi', example: 'ぎゅうにく (beef)' },
      { character: 'ぐ', romaji: 'gu', example: 'ぐんじ (military)' },
      { character: 'げ', romaji: 'ge', example: 'げつようび (Monday)' },
      { character: 'ご', romaji: 'go', example: 'ごはん (rice)' }
    ]},
    { row: 'ざ行', chars: [
      { character: 'ざ', romaji: 'za', example: 'ざっし (magazine)' },
      { character: 'じ', romaji: 'ji', example: 'じかん (time)' },
      { character: 'ず', romaji: 'zu', example: 'みず (water)' },
      { character: 'ぜ', romaji: 'ze', example: 'ぜんぶ (all)' },
      { character: 'ぞ', romaji: 'zo', example: 'ぞう (elephant)' }
    ]},
    { row: 'だ行', chars: [
      { character: 'だ', romaji: 'da', example: 'だいがく (university)' },
      { character: 'ぢ', romaji: 'di', example: 'はなぢ (nosebleed)' },
      { character: 'づ', romaji: 'du', example: 'つづく (continue)' },
      { character: 'で', romaji: 'de', example: 'でんしゃ (train)' },
      { character: 'ど', romaji: 'do', example: 'どうぶつ (animal)' }
    ]},
    { row: 'ば行', chars: [
      { character: 'ば', romaji: 'ba', example: 'ばら (rose)' },
      { character: 'び', romaji: 'bi', example: 'びじゅつ (art)' },
      { character: 'ぶ', romaji: 'bu', example: 'ぶた (pig)' },
      { character: 'べ', romaji: 'be', example: 'べんきょう (study)' },
      { character: 'ぼ', romaji: 'bo', example: 'ぼうし (hat)' }
    ]},
    { row: 'ぱ行', chars: [
      { character: 'ぱ', romaji: 'pa', example: 'ぱん (bread)' },
      { character: 'ぴ', romaji: 'pi', example: 'ぴんく (pink)' },
      { character: 'ぷ', romaji: 'pu', example: 'ぷーる (pool)' },
      { character: 'ぺ', romaji: 'pe', example: 'ぺん (pen)' },
      { character: 'ぽ', romaji: 'po', example: 'ぽすと (post)' }
    ]}
  ]
};

interface Props {
  onCharacterClick: (char: any) => void;
  playingAudio: string | null;
}

const HiraganaChart: React.FC<Props> = ({ onCharacterClick, playingAudio }) => {
  const [hoveredChar, setHoveredChar] = useState<string | null>(null);
  const [showRomaji, setShowRomaji] = useState(true);
  const [learnedChars, setLearnedChars] = useState<Set<string>>(new Set());

  const toggleLearned = useCallback((character: string) => {
    setLearnedChars(prev => {
      const newLearned = new Set(prev);
      if (newLearned.has(character)) {
        newLearned.delete(character);
      } else {
        newLearned.add(character);
      }
      return newLearned;
    });
  }, []);

  const handleCharacterHover = useCallback((character: string | null) => {
    setHoveredChar(character);
  }, []);

  const renderSection = useCallback((data: any[], title: string, isDakuten = false) => (
    <div className={styles.tableSection}>
      <div className={styles.sectionHeader}>
        <h3 className={styles.sectionTitle}>
          <span className={styles.titleIcon}>{isDakuten ? 'が' : 'ひ'}</span>
          {title}
          <div className={styles.charCount}>
            {learnedChars.size} / {isDakuten ? '25' : '46'} learned
          </div>
        </h3>
        
        <div className={styles.controls}>
          <button 
            className={`${styles.controlBtn} ${!showRomaji ? styles.active : ''}`}
            onClick={() => setShowRomaji(!showRomaji)}
            aria-label="Toggle Romaji visibility"
          >
            {showRomaji ? 'Hide Romaji' : 'Show Romaji'}
          </button>
        </div>
      </div>

      <div className={styles.translucent_container}>
        <div className={styles.tableBody}>
          {data.map((rowData, rowIndex) => (
            <div key={rowIndex} className={styles.tableRow}>
              {rowData.chars.map((char: any, colIndex: number) => (
                <div key={colIndex} className={styles.charCell}>
                  {char.character && (
                    <div 
                      className={`${styles.char_card} ${
                        playingAudio === char.character ? styles.playing : ''
                      } ${learnedChars.has(char.character) ? styles.learned : ''} ${
                        isDakuten ? styles.dakuten : ''
                      }`}
                      onMouseEnter={() => handleCharacterHover(char.character)}
                      onMouseLeave={() => handleCharacterHover(null)}
                      onClick={() => onCharacterClick(char)}
                      onDoubleClick={() => toggleLearned(char.character)}
                      role="button"
                      tabIndex={0}
                      aria-label={`Japanese character ${char.character}, pronounced ${char.romaji}`}
                    >
                      {/* Translucent main container */}
                      <div className={styles.translucent_box}>
                        {/* Small box at bottom with English character */}
                        <div className={`${styles.romaji_box} ${!showRomaji ? styles.hidden : ''}`}>
                          {char.romaji}
                        </div>
                        
                        {/* Large box with Japanese character */}
                        <div className={styles.japanese_box}>
                          <span className={styles.japanese_character}>
                            {char.character}
                          </span>
                        </div>
                      </div>
                      
                      {/* Example tooltip */}
                      {hoveredChar === char.character && char.example && (
                        <div className={styles.example_tooltip}>
                          {char.example}
                        </div>
                      )}
                      
                      {/* Action icons */}
                      <div className={styles.action_icons}>
                        <button 
                          className={styles.audio_btn} 
                          onClick={(e) => {
                            e.stopPropagation();
                            onCharacterClick(char);
                          }}
                          aria-label="Play audio"
                        >
                          <svg viewBox="0 0 24 24" fill="currentColor">
                            <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02z"/>
                          </svg>
                        </button>
                        
                        {learnedChars.has(char.character) && (
                          <div className={styles.learned_badge}>
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
  ), [learnedChars, showRomaji, hoveredChar, playingAudio, onCharacterClick, toggleLearned, handleCharacterHover]);

  return (
    <div className={styles.hiragana_chart}>
      {renderSection(hiraganaData.basic, 'Basic Hiragana (Gojūon)')}
      {renderSection(hiraganaData.dakuten, 'Dakuten (濁点) & Handakuten (半濁点)', true)}
    </div>
  );
};

export default HiraganaChart;
