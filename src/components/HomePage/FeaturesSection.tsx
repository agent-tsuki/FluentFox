// components/FeaturesSection.tsx
import React, { useState } from 'react';
import styles from './FeaturesSection.module.css';

const features = [
  {
    icon: '🔤',
    title: 'Master the Alphabet',
    description: 'Learn Hiragana and Katakana with interactive stroke order, audio pronunciation, and typing practice.',
    color: '#667eea'
  },
  {
    icon: '🀄',
    title: 'Kanji Made Simple',
    description: 'Build your kanji knowledge with radicals, mnemonics, and JLPT-organized lessons from N5 to N1.',
    color: '#764ba2'
  },
  {
    icon: '📚',
    title: 'Grammar That Clicks',
    description: 'Understand Japanese grammar through clear patterns, practical examples, and interactive exercises.',
    color: '#f093fb'
  },
  {
    icon: '🎯',
    title: 'JLPT Success Path',
    description: 'Structured preparation for all JLPT levels with mock tests, progress tracking, and exam strategies.',
    color: '#f5576c'
  },
  {
    icon: '🤖',
    title: 'AI-Powered Learning',
    description: 'Get instant feedback on writing, pronunciation, and grammar with our advanced AI tutor.',
    color: '#4facfe'
  },
  {
    icon: '🎮',
    title: 'Gamified Practice',
    description: 'Stay motivated with streaks, achievements, and fun mini-games that make learning addictive.',
    color: '#43e97b'
  }
];

const FeaturesSection: React.FC = () => {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);

  return (
    <section className={styles.features}>
      <div className={styles.container}>
        <div className={styles.sectionHeader}>
          <h2 className={styles.title}>
            Why FluentFox <span className={styles.gradient}>Works</span>
          </h2>
          <p className={styles.subtitle}>
            Everything you need to master Japanese, all in one intelligent platform
          </p>
        </div>
        
        <div className={styles.featuresGrid}>
          {features.map((feature, index) => (
            <div
              key={index}
              className={`${styles.featureCard} ${hoveredIndex === index ? styles.hovered : ''}`}
              onMouseEnter={() => setHoveredIndex(index)}
              onMouseLeave={() => setHoveredIndex(null)}
              style={{ '--accent-color': feature.color } as React.CSSProperties}
            >
              <div className={styles.cardIcon}>
                <span>{feature.icon}</span>
              </div>
              <h3 className={styles.cardTitle}>{feature.title}</h3>
              <p className={styles.cardDescription}>{feature.description}</p>
              <div className={styles.cardGlow}></div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default FeaturesSection;
