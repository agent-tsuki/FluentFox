// AlphabetChart.tsx
import React, { useState } from 'react';
import HiraganaChart from './Hiragana/HiraganaChart.tsx';
import KatakanaChart from './Katakana/KatakanaChart.tsx';
import CharacterDetail from './CharacterDetail.tsx';
import PracticeMode from './PracticeMode.tsx';
import styles from './AlphabetChart.module.css';

interface Character {
  character: string;
  romaji: string;
  audio?: string;
  strokeOrder?: string[];
  mnemonics?: string;
}

const AlphabetChart: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'hiragana' | 'katakana'>('hiragana');
  const [selectedCharacter, setSelectedCharacter] = useState<Character | null>(null);
  const [practiceMode, setPracticeMode] = useState(false);
  const [playingAudio, setPlayingAudio] = useState<string | null>(null);

  const playAudio = (character: string) => {
    setPlayingAudio(character);
    // Simulate audio play - in real implementation, load actual audio files
    setTimeout(() => setPlayingAudio(null), 1000);
  };

  const handleCharacterClick = (character: Character) => {
    setSelectedCharacter(character);
    playAudio(character.character);
  };

  return (
    <div className={styles.alphabetPage}>
      {/* Header Section */}
      <section className={styles.header}>
        <div className="container">
          <div className={styles.headerContent}>
            <div className={styles.breadcrumb}>
              <a href="/">Home</a>
              <span className={styles.separator}>›</span>
              <a href="/alphabet">Alphabet</a>
              <span className={styles.separator}>›</span>
              <span>Hiragana & Katakana</span>
            </div>
            
            <h1 className={styles.title}>
              Japanese <span className="gradient-text">Alphabet Charts</span>
            </h1>
            
            <p className={styles.subtitle}>
              Master the foundation of Japanese with interactive Hiragana and Katakana charts. 
              Click any character to hear pronunciation and see stroke order.
            </p>
            
            <div className={styles.stats}>
              <div className={styles.stat}>
                <span className={styles.statNumber}>46</span>
                <span className={styles.statLabel}>Basic Characters</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statNumber}>25</span>
                <span className={styles.statLabel}>Combination Sounds</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statNumber}>5</span>
                <span className={styles.statLabel}>Vowels</span>
              </div>
            </div>
          </div>
          
          <div className={styles.mascot}>
            <div className={styles.foxAvatar}>
              🦊
              <div className={styles.speechBubble}>
                <span>Let's learn together!</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Controls Section */}
      <section className={styles.controls}>
        <div className="container">
          <div className={styles.controlsContent}>
            <div className={styles.tabButtons}>
              <button
                className={`${styles.tabButton} ${activeTab === 'hiragana' ? styles.active : ''}`}
                onClick={() => setActiveTab('hiragana')}
              >
                <span className={styles.tabIcon}>ひ</span>
                Hiragana
              </button>
              <button
                className={`${styles.tabButton} ${activeTab === 'katakana' ? styles.active : ''}`}
                onClick={() => setActiveTab('katakana')}
              >
                <span className={styles.tabIcon}>カ</span>
                Katakana
              </button>
            </div>
            
            <div className={styles.actionButtons}>
              <button
                className={`${styles.actionBtn} ${practiceMode ? styles.active : ''}`}
                onClick={() => setPracticeMode(!practiceMode)}
              >
                <span className={styles.btnIcon}>🎯</span>
                {practiceMode ? 'Exit Practice' : 'Practice Mode'}
              </button>
              <button className={styles.actionBtn}>
                <span className={styles.btnIcon}>🔊</span>
                Play All
              </button>
              <button className={styles.actionBtn}>
                <span className={styles.btnIcon}>📱</span>
                Download PDF
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* Main Chart Section */}
      <section className={styles.chartSection}>
        <div className="container">
          {practiceMode ? (
            <PracticeMode 
              type={activeTab}
              onExit={() => setPracticeMode(false)}
            />
          ) : (
            <div className={styles.chartContainer}>
              {activeTab === 'hiragana' ? (
                <HiraganaChart
                  onCharacterClick={handleCharacterClick}
                  playingAudio={playingAudio}
                />
              ) : (
                <KatakanaChart
                  onCharacterClick={handleCharacterClick}
                  playingAudio={playingAudio}
                />
              )}
            </div>
          )}
        </div>
      </section>

      {/* Character Detail Modal */}
      {selectedCharacter && (
        <CharacterDetail
          character={selectedCharacter}
          type={activeTab}
          onClose={() => setSelectedCharacter(null)}
        />
      )}
    </div>
  );
};

export default AlphabetChart;
