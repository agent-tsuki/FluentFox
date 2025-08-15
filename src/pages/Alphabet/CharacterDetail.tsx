// components/CharacterDetail.tsx
import React, { useState } from 'react';
import styles from './CharacterDetail.module.css';

interface Props {
  character: {
    character: string;
    romaji: string;
  };
  type: 'hiragana' | 'katakana';
  onClose: () => void;
}

const CharacterDetail: React.FC<Props> = ({ character, type, onClose }) => {
  const [activeTab, setActiveTab] = useState<'pronunciation' | 'stroke' | 'practice'>('pronunciation');

  return (
    <div className={styles.overlay} onClick={onClose}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <button className={styles.closeBtn} onClick={onClose}>
          <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M18 6L6 18M6 6l12 12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
        </button>
        
        <div className={styles.header}>
          <div className={styles.charDisplay}>
            <span className={styles.bigChar}>{character.character}</span>
            <div className={styles.charInfo}>
              <span className={styles.romaji}>{character.romaji}</span>
              <span className={styles.type}>
                {type === 'hiragana' ? 'Hiragana' : 'Katakana'}
              </span>
            </div>
          </div>
          
          <button className={styles.playBtn}>
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M8 5v14l11-7z"/>
            </svg>
            Play Sound
          </button>
        </div>
        
        <div className={styles.tabs}>
          <button
            className={`${styles.tab} ${activeTab === 'pronunciation' ? styles.active : ''}`}
            onClick={() => setActiveTab('pronunciation')}
          >
            <span className={styles.tabIcon}>🔊</span>
            Pronunciation
          </button>
          <button
            className={`${styles.tab} ${activeTab === 'stroke' ? styles.active : ''}`}
            onClick={() => setActiveTab('stroke')}
          >
            <span className={styles.tabIcon}>✏️</span>
            Stroke Order
          </button>
          <button
            className={`${styles.tab} ${activeTab === 'practice' ? styles.active : ''}`}
            onClick={() => setActiveTab('practice')}
          >
            <span className={styles.tabIcon}>📝</span>
            Practice
          </button>
        </div>
        
        <div className={styles.content}>
          {activeTab === 'pronunciation' && (
            <div className={styles.pronunciationContent}>
              <div className={styles.audioSection}>
                <div className={styles.waveform}>
                  <div className={styles.wave}></div>
                  <div className={styles.wave}></div>
                  <div className={styles.wave}></div>
                  <div className={styles.wave}></div>
                  <div className={styles.wave}></div>
                </div>
                <p className={styles.tip}>
                  Click the character or play button to hear the pronunciation
                </p>
              </div>
              
              <div className={styles.examples}>
                <h4>Example Words</h4>
                <div className={styles.wordList}>
                  <div className={styles.word}>
                    <span className={styles.japanese}>あり</span>
                    <span className={styles.meaning}>ant</span>
                  </div>
                  <div className={styles.word}>
                    <span className={styles.japanese}>あさ</span>
                    <span className={styles.meaning}>morning</span>
                  </div>
                </div>
              </div>
            </div>
          )}
          
          {activeTab === 'stroke' && (
            <div className={styles.strokeContent}>
              <div className={styles.strokeDisplay}>
                <div className={styles.strokeChar}>{character.character}</div>
                <div className={styles.strokeSteps}>
                  <div className={styles.step}>1</div>
                  <div className={styles.step}>2</div>
                  <div className={styles.step}>3</div>
                </div>
              </div>
              <button className={styles.animateBtn}>
                <span className={styles.btnIcon}>▶️</span>
                Play Animation
              </button>
              <p className={styles.tip}>
                Follow the stroke order to write the character correctly
              </p>
            </div>
          )}
          
          {activeTab === 'practice' && (
            <div className={styles.practiceContent}>
              <div className={styles.practiceArea}>
                <canvas className={styles.drawingCanvas}></canvas>
                <div className={styles.practiceControls}>
                  <button className={styles.clearBtn}>Clear</button>
                  <button className={styles.checkBtn}>Check</button>
                </div>
              </div>
              <p className={styles.tip}>
                Practice writing the character in the area above
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default CharacterDetail;
