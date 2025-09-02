// components/LearningPathSection.tsx
import React, { useState, useRef, useEffect } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import styles from '../../styles/pages/LearningPathSection.module.css';

// Register GSAP plugins
gsap.registerPlugin(ScrollTrigger);

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
  const sectionRef = useRef<HTMLElement>(null);
  const progressLineRef = useRef<HTMLDivElement>(null);
  const headerRef = useRef<HTMLDivElement>(null);
  const cardsRef = useRef<(HTMLDivElement | null)[]>([]);
  const dotsRef = useRef<(HTMLDivElement | null)[]>([]);
  const ctaRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const ctx = gsap.context(() => {
      // Set initial states with proper timing
      gsap.set([headerRef.current, ctaRef.current], { opacity: 0, y: 50 });
      gsap.set(cardsRef.current, { opacity: 0, y: 100, rotationY: -15 });
      gsap.set(dotsRef.current, { scale: 0, rotation: -180 });

      // Smooth header animation with better timing
      gsap.to(headerRef.current, {
        opacity: 1,
        y: 0,
        duration: 1.2,
        ease: "power2.out",
        scrollTrigger: {
          trigger: sectionRef.current,
          start: "top 80%",
          toggleActions: "play none none reverse"
        }
      });

      // FIXED: Progress line that grows downward on scroll
      gsap.set(progressLineRef.current, { 
        height: "0%", 
        opacity: 1
      });

      gsap.to(progressLineRef.current, {
        height: "100%",
        duration: 1,
        ease: "none", // Critical for smooth scrub
        scrollTrigger: {
          trigger: sectionRef.current,
          start: "top 60%",
          end: "bottom 20%",
          scrub: 1.5, // Smoother scrub value
          onUpdate: (self) => {
            // Smooth color transition based on progress
            const progress = self.progress;
            const hue = 240 - (progress * 120); // Blue to green transition
            gsap.set(progressLineRef.current, {
              background: `linear-gradient(180deg, 
                hsl(${hue}, 70%, 60%) 0%, 
                hsl(${hue + 20}, 80%, 65%) 50%, 
                hsl(${hue + 40}, 90%, 70%) 100%
              )`
            });
          },
          onEnter: () => {
            gsap.to(progressLineRef.current, { 
              opacity: 1, 
              duration: 0.5,
              ease: "power2.out"
            });
          }
        }
      });

      // Enhanced dots animation that follows progress smoothly
      dotsRef.current.forEach((dot, index) => {
        if (!dot) return;

        // Individual dot reveal animation
        gsap.to(dot, {
          scale: 1,
          rotation: 0,
          duration: 0.6,
          ease: "back.out(1.7)",
          scrollTrigger: {
            trigger: dot,
            start: "top 85%",
            toggleActions: "play none none reverse"
          }
        });

        // Smooth progress-based dot highlighting that follows line height
        ScrollTrigger.create({
          trigger: sectionRef.current,
          start: "top 60%",
          end: "bottom 20%",
          scrub: 1,
          onUpdate: (self) => {
            const progress = self.progress;
            const totalDots = dotsRef.current.length;
            const dotThreshold = (index + 1) / totalDots; // When this dot should activate
            
            if (progress >= dotThreshold - 0.1) {
              gsap.to(dot, {
                backgroundColor: "#10b981",
                borderColor: "#10b981",
                boxShadow: "0 8px 25px rgba(16, 185, 129, 0.4)",
                scale: 1.2,
                duration: 0.3,
                ease: "power2.out"
              });
            } else {
              gsap.to(dot, {
                backgroundColor: "rgba(99, 102, 241, 0.3)",
                borderColor: "#e5e7eb",
                boxShadow: "0 4px 15px rgba(0,0,0,0.1)",
                scale: 1,
                duration: 0.3,
                ease: "power2.out"
              });
            }
          }
        });
      });

      // Smooth staggered card animations
      cardsRef.current.forEach((card, index) => {
        if (!card) return;

        // Enhanced card animation timeline
        const tl = gsap.timeline({
          scrollTrigger: {
            trigger: card,
            start: "top 80%",
            toggleActions: "play none none reverse"
          }
        });

        tl.to(card, {
          opacity: 1,
          x: 0,
          y: 0,
          rotationY: 0,
          scale: 1,
          duration: 0.8,
          ease: "power2.out"
        })
        .to(card.querySelector(`.${styles.progressFill}`), {
          width: `${pathSteps[index].progress}%`,
          duration: 1.5,
          ease: "power2.out"
        }, "-=0.4");

        // Smooth hover animations with better performance
        let hoverTween: gsap.core.Tween | null = null;
        
        card.addEventListener('mouseenter', () => {
          if (hoverTween) hoverTween.kill();
          hoverTween = gsap.to(card, {
            y: -15,
            scale: 1.03,
            rotationX: 3,
            boxShadow: "0 20px 60px rgba(99, 102, 241, 0.25)",
            duration: 0.3,
            ease: "power2.out"
          });
        });

        card.addEventListener('mouseleave', () => {
          if (hoverTween) hoverTween.kill();
          hoverTween = gsap.to(card, {
            y: 0,
            scale: 1,
            rotationX: 0,
            boxShadow: "0 8px 30px rgba(0, 0, 0, 0.1)",
            duration: 0.3,
            ease: "power2.out"
          });
        });

        // Improved click feedback
        card.addEventListener('click', () => {
          gsap.to(card, {
            scale: 0.98,
            duration: 0.1,
            yoyo: true,
            repeat: 1,
            ease: "power2.inOut"
          });
        });
      });

      // Smooth CTA animation
      gsap.to(ctaRef.current, {
        opacity: 1,
        y: 0,
        duration: 1,
        ease: "power2.out",
        scrollTrigger: {
          trigger: ctaRef.current,
          start: "top 85%",
          toggleActions: "play none none reverse"
        }
      });

      // Subtle parallax background effect
      gsap.to(sectionRef.current, {
        backgroundPosition: "50% 80%",
        ease: "none",
        scrollTrigger: {
          trigger: sectionRef.current,
          start: "top bottom",
          end: "bottom top",
          scrub: 2 // Slower, smoother parallax
        }
      });

    }, sectionRef);

    return () => ctx.revert(); // Cleanup
  }, []);

  const handleStepClick = (index: number) => {
    setActiveStep(index);
    
    // Enhanced click feedback with GSAP
    const card = cardsRef.current[index];
    if (card) {
      gsap.to(card, {
        backgroundColor: "rgba(99, 102, 241, 0.15)",
        borderColor: "#6366f1",
        duration: 0.3,
        ease: "power2.out"
      });

      // Reset other cards smoothly
      cardsRef.current.forEach((otherCard, otherIndex) => {
        if (otherIndex !== index && otherCard) {
          gsap.to(otherCard, {
            backgroundColor: "rgba(255, 255, 255, 0.95)",
            borderColor: "rgba(255, 255, 255, 0.3)",
            duration: 0.3,
            ease: "power2.out"
          });
        }
      });
    }

    // Haptic feedback for mobile
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
        <div ref={headerRef} className={styles.sectionHeader}>
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
            />
            
            {/* Enhanced Progress Particles */}
            <div className={styles.progressParticles}>
              <div className={styles.particle} style={{ animationDelay: '0s' }}></div>
              <div className={styles.particle} style={{ animationDelay: '1.3s' }}></div>
              <div className={styles.particle} style={{ animationDelay: '2.6s' }}></div>
            </div>
            
            {/* Progress indicator text */}
            <div className={styles.progressIndicator}>
              <span>Scroll to see progress</span>
            </div>
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
              >
                {/* Enhanced Connector Dot */}
                <div 
                  ref={el => { dotsRef.current[index] = el; }}
                  className={`${styles.connectorDot} ${styles[status]}`}
                />
                
                {/* Enhanced Step Card */}
                <div
                  ref={el => { cardsRef.current[index] = el; }}
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
                  {/* Animated background highlight */}
                  <div className={styles.cardHighlight}></div>
                  
                  {/* Step Header */}
                  <div className={styles.stepHeader}>
                    <div className={styles.stepNumber}>{index + 1}</div>
                    <div className={styles.stepIcon}>{step.icon}</div>
                  </div>
                  
                  {/* Step Content */}
                  <div className={styles.stepContent}>
                    <div className={styles.stepLevel}>{step.level}</div>
                    <h3 className={styles.stepTitle}>{step.title}</h3>
                    <p className={styles.stepDescription}>{step.description}</p>
                    
                    {/* Enhanced Progress Bar */}
                    <div className={styles.progressBar}>
                      <div 
                        className={styles.progressFill}
                        style={{ width: '0%' }} // Will be animated by GSAP
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
                  
                  {/* Enhanced 3D Shadow Effect */}
                  <div className={styles.cardShadow} />
                </div>
              </div>
            );
          })}
        </div>

        {/* Enhanced Call to Action */}
        <div ref={ctaRef} className={styles.pathCTA}>
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
