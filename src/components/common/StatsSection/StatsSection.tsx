import React from 'react';
import styles from './StatsSection.module.css';

const StatsSection: React.FC = () => {
  const stats = [
    { number: '2,000+', label: 'Kanji Characters' },
    { number: '6,000+', label: 'Vocabulary Words' },
    { number: '50,000+', label: 'Active Learners' },
    { number: '95%', label: 'Success Rate' },
  ];

  return (
    <section className={styles.stats}>
      <div className={styles.container}>
        <div className={styles.statsGrid}>
          {stats.map((stat, index) => (
            <div key={index} className={styles.statItem}>
              <div className={styles.statNumber}>{stat.number}</div>
              <div className={styles.statLabel}>{stat.label}</div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default StatsSection;
