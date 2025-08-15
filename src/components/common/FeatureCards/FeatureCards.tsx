
import React from 'react';
import styles from './FeatureCards.module.css';

const FeatureCards: React.FC = () => {
  const features = [
    {
      title: 'Smart Mnemonics',
      description: 'Memorable stories and visual associations make kanji stick in your memory permanently.',
      icon: '🧠'
    },
    {
      title: 'Spaced Repetition',
      description: 'Our SRS algorithm shows you cards at the perfect time to maximize retention.',
      icon: '🔄'
    },
    {
      title: 'Progressive Learning',
      description: 'Start with radicals, build to kanji, then master vocabulary in logical order.',
      icon: '📈'
    },
    {
      title: 'JLPT Focused',
      description: 'Aligned with JLPT levels N5-N1 to support your certification goals.',
      icon: '🎯'
    },
    {
      title: 'Audio Pronunciation',
      description: 'Native speaker audio for every word helps perfect your pronunciation.',
      icon: '🔊'
    },
    {
      title: 'Progress Tracking',
      description: 'Detailed analytics show your learning journey and areas for improvement.',
      icon: '📊'
    },
  ];

  return (
    <section className={styles.features}>
      <div className={styles.container}>
        <div className={styles.header}>
          <h2 className={styles.title}>Why Choose FluentFox?</h2>
          <p className={styles.subtitle}>
            Our scientifically-proven method makes learning Japanese faster and more enjoyable
          </p>
        </div>
        <div className={styles.featureGrid}>
          {features.map((feature, index) => (
            <div key={index} className={styles.featureCard}>
              <div className={styles.featureIcon}>{feature.icon}</div>
              <h3 className={styles.featureTitle}>{feature.title}</h3>
              <p className={styles.featureDescription}>{feature.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default FeatureCards;
