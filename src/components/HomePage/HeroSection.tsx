// components/HeroSection.tsx
import React, { useState, useEffect } from 'react';
import styles from '../../styles/pages/HeroSection.module.css';

const HeroSection: React.FC = () => {
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    setIsVisible(true);
  }, []);

  return (
    <section className={styles.hero}>
      <div className={styles.container}>
        <div className={styles.heroContent}>
          <div className={styles.heroText}>
            <div className={`${styles.badge} ${isVisible ? styles.fadeIn : ''}`}>
              <span className={styles.badgeIcon}>🎉</span>
              New: AI-powered grammar correction
            </div>
            
            <h1 className={`${styles.title} ${isVisible ? styles.slideUp : ''}`}>
              Master Japanese the 
              <span className={styles.gradient}> Smart Way</span>
            </h1>
            
            <p className={`${styles.subtitle} ${isVisible ? styles.slideUp : ''}`}>
              From Hiragana to JLPT N1 — Learn with structure, have fun, and see real results. 
              Join thousands transforming their Japanese journey.
            </p>
            
            <div className={`${styles.heroActions} ${isVisible ? styles.slideUp : ''}`}>
              <button className={styles.primaryBtn}>
                <span>Start Free Trial</span>
                <svg className={styles.arrow} viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clipRule="evenodd" />
                </svg>
              </button>
              <button className={styles.secondaryBtn}>
                Take Placement Test
              </button>
            </div>
            
            <div className={styles.trustIndicators}>
              <div className={styles.indicator}>
                <span className={styles.indicatorIcon}>⭐</span>
                <span>4.9/5 rating</span>
              </div>
              <div className={styles.indicator}>
                <span className={styles.indicatorIcon}>👥</span>
                <span>50,000+ learners</span>
              </div>
              <div className={styles.indicator}>
                <span className={styles.indicatorIcon}>🎯</span>
                <span>JLPT focused</span>
              </div>
            </div>
          </div>
          
          <div className={styles.heroVisual}>
            <div className={styles.foxContainer}>
              <div className={styles.foxAvatar}>
                🦊
                <div className={styles.speechBubble}>
                  <span>がんばって！</span>
                </div>
              </div>
              <div className={styles.floatingElements}>
                <div className={styles.floatingCard} data-delay="0">
                  <span>ひらがな</span>
                </div>
                <div className={styles.floatingCard} data-delay="1">
                  <span>漢字</span>
                </div>
                <div className={styles.floatingCard} data-delay="2">
                  <span>文法</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};

export default HeroSection;
