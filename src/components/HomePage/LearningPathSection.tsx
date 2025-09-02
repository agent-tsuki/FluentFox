// components/LearningPathSection.tsx
import React, { useState, useRef, useEffect } from 'react';
import styles from '../../styles/pages/LearningPathSection.module.css';

const pathSteps = [
  { level: 'Beginner', title: 'Hiragana & Katakana', description: 'Master the Japanese alphabets', progress: 100, icon: '🔤' },
  { level: 'Elementary', title: 'Basic Grammar', description: 'Core sentence patterns', progress: 75, icon: '📝' },
  { level: 'Pre-Intermediate', title: 'Essential Kanji', description: '300+ most common characters', progress: 60, icon: '🀄' },
  { level: 'Intermediate', title: 'JLPT N4-N3', description: 'Conversational fluency', progress: 40, icon: '💬' },
  { level: 'Upper-Intermediate', title: 'JLPT N2', description: 'Advanced grammar & vocab', progress: 20, icon: '📚' },
  { level: 'Advanced', title: 'JLPT N1', description: 'Near-native proficiency', progress: 0, icon: '🎓' }
];

const LearningPathSection: React.FC = () => {
  const [activeStep, setActiveStep] = useState(0);
  const [scrollProgress, setScrollProgress] = useState(0);
  const [isVisible, setIsVisible] = useState(false);
  const sectionRef = useRef<HTMLElement>(null);
  const progressLineRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleScroll = () => {
      if (sectionRef.current) {
        const rect = sectionRef.current.getBoundingClientRect();
        const windowHeight = window.innerHeight;
        
        // Calculate visibility percentage
        const elementTop = rect.top;
        const elementHeight = rect.height;
        const visibleHeight = Math.min(windowHeight - Math.max(elementTop, 0), elementHeight);
        const visibilityRatio = Math.max(0, visibleHeight / windowHeight);
        
        setScrollProgress(Math.min(visibilityRatio * 1.2, 1));
        setIsVisible(visibilityRatio > 0.1);
      }
    };

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsVisible(true);
        }
      },
      { threshold: 0.1 }
    );

    if (sectionRef.current) {
      observer.observe(sectionRef.current);
    }

    window.addEventListener('scroll', handleScroll);
    handleScroll(); // Initial call

    return () => {
      observer.disconnect();
      window.removeEventListener('scroll', handleScroll);
    };
  }, []);

  const handleStepClick = (index: number) => {
    setActiveStep(index);
    
    // Add a subtle haptic feedback (if supported)
    if (navigator.vibrate) {
      navigator.vibrate(50);
    }
  };

  const getStepStatus = (index: number) => {
    const step = pathSteps[index];
    if (step.progress === 100) return 'completed';
    if (step.progress > 0) return 'in-progress';
    return 'locked';
  };

  return (
    <section ref={sectionRef} className={styles.learningPath}>
      <div className={styles.container}>
        {/* Section Header */}
        <div className={styles.sectionHeader}>
          <h2 className={styles.title}>
            Your <span className={styles.gradient}>Learning Journey</span>
          </h2>
          <p className={styles.subtitle}>
            Interactive roadmap to Japanese mastery - click any step to explore
          </p>
        </div>

        {/* Progress Line Container */}
        <div className={styles.progressContainer}>
          <div className={styles.progressTrack}>
            <div 
              ref={progressLineRef}
              className={styles.progressLine}
              style={{ 
                transform: `scaleX(${scrollProgress})`,
                opacity: isVisible ? 1 : 0 
              }}
            />
          </div>
        </div>

        {/* Learning Path Steps */}
        <div className={styles.pathGrid}>
          {pathSteps.map((step, index) => {
            const isActive = activeStep === index;
            const status = getStepStatus(index);
            const isLeft = index % 2 === 0;
            
            return (
              <div
                key={index}
                className={`${styles.stepWrapper} ${isLeft ? styles.left : styles.right}`}
                style={{ 
                  animationDelay: `${index * 0.15}s`,
                  opacity: isVisible ? 1 : 0 
                }}
              >
                {/* Connector Dot */}
                <div className={`${styles.connectorDot} ${status}`} />
                
                {/* Step Card */}
                <div
                  className={`${styles.stepCard} ${isActive ? styles.active : ''} ${styles[status]}`}
                  onClick={() => handleStepClick(index)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault();
                      handleStepClick(index);
                    }
                  }}
                  tabIndex={0}
                  role="button"
                  aria-pressed={isActive}
                  aria-label={`Step ${index + 1}: ${step.title}`}
                >
                  {/* Step Number & Icon */}
                  <div className={styles.stepHeader}>
                    <div className={styles.stepNumber}>{index + 1}</div>
                    <div className={styles.stepIcon}>{step.icon}</div>
                  </div>

                  {/* Step Content */}
                  <div className={styles.stepContent}>
                    <div className={styles.stepLevel}>{step.level}</div>
                    <h3 className={styles.stepTitle}>{step.title}</h3>
                    <p className={styles.stepDescription}>{step.description}</p>
                    
                    {/* Progress Bar */}
                    <div className={styles.progressBar}>
                      <div 
                        className={styles.progressFill}
                        style={{ width: `${step.progress}%` }}
                      />
                      <span className={styles.progressText}>{step.progress}%</span>
                    </div>
                  </div>

                  {/* Interactive Elements */}
                  <div className={styles.stepActions}>
                    {status === 'completed' && (
                      <span className={styles.completedBadge}>✓ Completed</span>
                    )}
                    {status === 'in-progress' && (
                      <button className={styles.continueBtn}>Continue</button>
                    )}
                    {status === 'locked' && (
                      <span className={styles.lockedBadge}>🔒 Locked</span>
                    )}
                  </div>

                  {/* 3D Shadow Effect */}
                  <div className={styles.cardShadow} />
                </div>
              </div>
            );
          })}
        </div>

        {/* Call to Action */}
        <div className={styles.pathCTA}>
          <button className={styles.startBtn}>
            <span className={styles.btnText}>Continue Your Journey</span>
            <span className={styles.btnIcon}>🚀</span>
          </button>
          <div className={styles.encouragement}>
            <span>You're making great progress! Keep going! 💪</span>
          </div>
        </div>
      </div>
    </section>
  );
};

export default LearningPathSection;
