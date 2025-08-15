// AlphabetChart.tsx - Fixed & optimized version
import React, { useState, useCallback, lazy, Suspense } from 'react';
// Lazy load heavy chart components
const HiraganaChart = lazy(() => import('./Hiragana/HiraganaChart'));
const KatakanaChart = lazy(() => import('./Katakana/KatakanaChart'));
import CharacterDetail from './CharacterDetail';
import PracticeMode from './PracticeMode';
import styles from './AlphabetChart.module.css';

const AlphabetChart: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'hiragana' | 'katakana'>('hiragana');
  const [selectedCharacter, setSelectedCharacter] = useState<any>(null);
  const [practiceMode, setPracticeMode] = useState(false);
  const [playingAudio, setPlayingAudio] = useState<string | null>(null);

  const playAudio = useCallback((character: string) => {
    setPlayingAudio(character);
    // Simulate audio play duration
    window.setTimeout(() => setPlayingAudio(null), 1000);
  }, []);

  const handleCharacterClick = useCallback((character: any) => {
    setSelectedCharacter(character);
    if (character?.character) playAudio(character.character);
  }, [playAudio]);

  // Fixed tab switching function + reset transient UI state
  const handleTabChange = useCallback((tab: 'hiragana' | 'katakana') => {
    if (tab === activeTab) return;
    setActiveTab(tab);
    setSelectedCharacter(null);
    setPracticeMode(false);
    setPlayingAudio(null);
  }, [activeTab]);

  // Keyboard support for tab switcher
  const handleSwitcherKeyDown = useCallback((e: React.KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'ArrowRight' || e.key === 'ArrowLeft') {
      e.preventDefault();
      handleTabChange(activeTab === 'hiragana' ? 'katakana' : 'hiragana');
    }
  }, [activeTab, handleTabChange]);

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

      {/* Fixed Modern Controls Section */}
      <section className={styles.modernControls}>
        <div className="container">
          <div className={styles.controlsGrid}>
            {/* Fixed Tab Switcher */}
            <div className={styles.tabSwitcher}>
              <div
                className={styles.switcherTrack}
                role="tablist"
                aria-label="Alphabet selector"
                onKeyDown={handleSwitcherKeyDown}
              >
                <button
                  className={`${styles.switcherOption} ${activeTab === 'hiragana' ? styles.active : ''}`}
                  onClick={() => handleTabChange('hiragana')}
                  type="button"
                  role="tab"
                  aria-selected={activeTab === 'hiragana'}
                >
                  <span className={styles.optionIcon}>ひ</span>
                  <span className={styles.optionText}>Hiragana</span>
                </button>
                <button
                  className={`${styles.switcherOption} ${activeTab === 'katakana' ? styles.active : ''}`}
                  onClick={() => handleTabChange('katakana')}
                  type="button"
                  role="tab"
                  aria-selected={activeTab === 'katakana'}
                >
                  <span className={styles.optionIcon}>カ</span>
                  <span className={styles.optionText}>Katakana</span>
                </button>
                <div className={`${styles.activeIndicator} ${activeTab === 'katakana' ? styles.right : ''}`} />
              </div>
            </div>

            {/* Action Buttons */}
            <div className={styles.actionGroup}>
              <button
                className={`${styles.modernBtn} ${styles.practiceBtn} ${practiceMode ? styles.active : ''}`}
                onClick={() => setPracticeMode(!practiceMode)}
                type="button"
                aria-pressed={practiceMode}
              >
                <div className={styles.btnIcon}>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                    <path d="M12 2L2 7v10c0 5.55 3.84 10 9 11 5.16-1 9-5.45 9-11V7l-10-5z"/>
                    <path d="M9 12l2 2 4-4"/>
                  </svg>
                </div>
                <span className={styles.btnText}>Practice</span>
              </button>

              <button 
                className={`${styles.modernBtn} ${styles.playBtn}`}
                type="button"
                aria-label="Play all pronunciations"
              >
                <div className={styles.btnIcon}>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                    <polygon points="5,3 19,12 5,21"/>
                  </svg>
                </div>
                <span className={styles.btnText}>Play All</span>
              </button>

              <button 
                className={`${styles.modernBtn} ${styles.downloadBtn}`}
                type="button"
                aria-label="Download chart"
              >
                <div className={styles.btnIcon}>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                    <polyline points="7,10 12,15 17,10"/>
                    <line x1="12" y1="15" x2="12" y2="3"/>
                  </svg>
                </div>
                <span className={styles.btnText}>Download</span>
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* Chart Section */}
      <section className={styles.chartSection}>
        <div className="container">
          {practiceMode ? (
            <PracticeMode 
              type={activeTab}
              onExit={() => setPracticeMode(false)}
            />
          ) : (
            <div className={styles.modernChartContainer}>
              <Suspense fallback={<div className={styles.loading}>Loading chart…</div>}>
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
              </Suspense>
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
