// components/HowItWorksSection.tsx
import React from 'react';
import styles from './HowItWorksSection.module.css';

const steps = [
  {
    number: '01',
    title: 'Take Placement Test',
    description: 'Quick 5-minute assessment to determine your current level and create a personalized learning path.',
    icon: '📝',
    image: '/api/placeholder/300/200'
  },
  {
    number: '02',
    title: 'Follow Your Roadmap',
    description: 'Learn through interactive lessons, practice exercises, and AI-powered feedback tailored to your goals.',
    icon: '🗺️',
    image: '/api/placeholder/300/200'
  },
  {
    number: '03',
    title: 'Track Progress',
    description: 'Monitor your improvement with detailed analytics, achievement badges, and JLPT readiness scores.',
    icon: '📈',
    image: '/api/placeholder/300/200'
  }
];

const HowItWorksSection: React.FC = () => {
  return (
    <section className={styles.howItWorks}>
      <div className={styles.container}>
        <div className={styles.sectionHeader}>
          <h2 className={styles.title}>
            How <span className={styles.gradient}>FluentFox</span> Works
          </h2>
          <p className={styles.subtitle}>
            Simple, structured, and scientifically-backed approach to Japanese mastery
          </p>
        </div>
        
        <div className={styles.stepsContainer}>
          {steps.map((step, index) => (
            <div key={index} className={styles.step}>
              <div className={styles.stepContent}>
                <div className={styles.stepHeader}>
                  <div className={styles.stepNumber}>{step.number}</div>
                  <div className={styles.stepIcon}>{step.icon}</div>
                </div>
                <h3 className={styles.stepTitle}>{step.title}</h3>
                <p className={styles.stepDescription}>{step.description}</p>
              </div>
              <div className={styles.stepVisual}>
                <div className={styles.stepImagePlaceholder}>
                  <span>{step.icon}</span>
                </div>
              </div>
              {index < steps.length - 1 && (
                <div className={styles.stepConnector}>
                  <svg viewBox="0 0 24 24" fill="none">
                    <path d="M5 12H19M19 12L12 5M19 12L12 19" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                  </svg>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default HowItWorksSection;
