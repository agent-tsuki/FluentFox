// components/HeroSection.tsx
import React, { useState, useEffect, useCallback, useRef } from 'react';
import { motion, useAnimation, AnimatePresence } from 'framer-motion';
import styles from '../../styles/pages/HeroSection.module.css';

interface Bubble {
  id: number;
  x: number;
  y: number;
  size: number;
  speedX: number;
  speedY: number;
  color: string;
  opacity: number;
}

interface Particle {
  id: number;
  x: number;
  y: number;
  velocityX: number;
  velocityY: number;
  life: number;
  size: number;
  color: string;
}

const HeroSection: React.FC = () => {
  const [isVisible, setIsVisible] = useState(false);
  const [bubbles, setBubbles] = useState<Bubble[]>([]);
  const [particles, setParticles] = useState<Particle[]>([]);
  const [burstAnimations, setBurstAnimations] = useState<{[key: number]: boolean}>({});
  const foxControls = useAnimation();
  const containerRef = useRef<HTMLDivElement>(null);
  
  // Orbital word data with increased spacing
  const orbitalWords = {
    inner: ['ひらがな', '漢字', '文法'],
    middle: ['N5', 'N4', 'N3', 'N2', 'N1'],
    outer: ['リスニング', '読解', '会話', '作文']
  };

  const waterBubbleColors = [
    'rgba(59, 130, 246, 0.4)',
    'rgba(99, 102, 241, 0.4)', 
    'rgba(139, 92, 246, 0.4)',
    'rgba(6, 182, 212, 0.4)'
  ];

  // Create realistic water bubble
  const createBubble = useCallback((): Bubble => {
    const containerWidth = containerRef.current?.offsetWidth || 800;
    const containerHeight = containerRef.current?.offsetHeight || 600;
    
    return {
      id: Date.now() + Math.random(),
      x: Math.random() * (containerWidth - 100) + 50,
      y: containerHeight + 50,
      size: Math.random() * 60 + 30,
      speedX: (Math.random() - 0.5) * 1.5,
      speedY: -Math.random() * 3 - 2,
      color: waterBubbleColors[Math.floor(Math.random() * waterBubbleColors.length)],
      opacity: Math.random() * 0.5 + 0.5
    };
  }, []);

  // Enhanced firecracker burst with more particles
  const createBurstExplosion = useCallback((x: number, y: number, bubbleSize: number) => {
    const newParticles: Particle[] = [];
    const particleCount = Math.random() * 25 + 20; // More particles for bigger explosion
    
    for (let i = 0; i < particleCount; i++) {
      const angle = (Math.PI * 2 * i) / particleCount;
      const speed = Math.random() * 12 + 6; // Faster particles
      const size = Math.random() * 6 + 3;
      
      newParticles.push({
        id: Date.now() + Math.random() + i,
        x,
        y,
        velocityX: Math.cos(angle) * speed + (Math.random() - 0.5) * 4,
        velocityY: Math.sin(angle) * speed + (Math.random() - 0.5) * 4,
        life: 1,
        size,
        color: waterBubbleColors[Math.floor(Math.random() * waterBubbleColors.length)]
      });
    }
    
    setParticles(prev => [...prev, ...newParticles]);
  }, []);

  // Handle bubble burst with animation
  const handleBubbleBurst = useCallback((bubble: Bubble) => {
    // Set burst animation state
    setBurstAnimations(prev => ({ ...prev, [bubble.id]: true }));
    
    // Create explosion particles
    createBurstExplosion(bubble.x + bubble.size/2, bubble.y + bubble.size/2, bubble.size);
    
    // Remove bubble after burst animation
    setTimeout(() => {
      setBubbles(prev => prev.filter(b => b.id !== bubble.id));
      setBurstAnimations(prev => {
        const newState = { ...prev };
        delete newState[bubble.id];
        return newState;
      });
    }, 300);
  }, [createBurstExplosion]);

  // Animation loops
  useEffect(() => {
    setIsVisible(true);
    
    // Start fox drawing animation
    foxControls.start({
      pathLength: 1,
      transition: { duration: 3, ease: "easeInOut", delay: 0.5 }
    });

    // Bubble generation interval
    const bubbleInterval = setInterval(() => {
      setBubbles(prev => {
        if (prev.length < 10) {
          return [...prev, createBubble()];
        }
        return prev;
      });
    }, 1500);

    // Animation frame for physics
    const animate = () => {
      setBubbles(prev => prev.map(bubble => ({
        ...bubble,
        x: bubble.x + bubble.speedX,
        y: bubble.y + bubble.speedY,
        speedY: bubble.speedY + 0.05, // Slight gravity effect
      })).filter(bubble => bubble.y > -150));

      setParticles(prev => prev.map(particle => ({
        ...particle,
        x: particle.x + particle.velocityX,
        y: particle.y + particle.velocityY,
        velocityX: particle.velocityX * 0.99, // Air resistance
        velocityY: particle.velocityY * 0.99 + 0.4, // Gravity
        life: particle.life - 0.015,
        size: particle.size * 0.995 // Shrinking effect
      })).filter(particle => particle.life > 0));
      
      requestAnimationFrame(animate);
    };
    
    animate();

    return () => {
      clearInterval(bubbleInterval);
    };
  }, [foxControls, createBubble]);

  // Enhanced self-drawing fox SVG
  const FoxLogo = () => (
    <motion.div className={styles.foxLogoContainer}>
      <motion.svg
        width="120"
        height="120"
        viewBox="0 0 120 120"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        initial={{ scale: 0, rotate: -180 }}
        animate={{ scale: 1, rotate: 0 }}
        transition={{ duration: 1.2, delay: 0.3, type: "spring", stiffness: 100 }}
      >
        {/* Fox body with enhanced path */}
        <motion.path
          d="M60 30 L40 45 Q35 55 40 65 L60 95 L80 65 Q85 55 80 45 Z"
          stroke="url(#foxGradient)"
          strokeWidth="4"
          fill="none"
          strokeLinecap="round"
          strokeLinejoin="round"
          initial={{ pathLength: 0 }}
          animate={foxControls}
        />
        
        {/* Fox ears with animation */}
        <motion.path
          d="M40 45 L25 25 L45 35 M80 45 L95 25 L75 35"
          stroke="url(#foxGradient)"
          strokeWidth="4"
          fill="none"
          strokeLinecap="round"
          initial={{ pathLength: 0 }}
          animate={{ pathLength: 1 }}
          transition={{ duration: 2.5, delay: 1.8 }}
        />
        
        {/* Fox face details */}
        <motion.circle
          cx="52"
          cy="55"
          r="3"
          fill="#4f46e5"
          initial={{ scale: 0 }}
          animate={{ scale: 1 }}
          transition={{ duration: 0.4, delay: 3 }}
        />
        <motion.circle
          cx="68"
          cy="55"
          r="3"
          fill="#4f46e5"
          initial={{ scale: 0 }}
          animate={{ scale: 1 }}
          transition={{ duration: 0.4, delay: 3.2 }}
        />
        
        {/* Enhanced nose */}
        <motion.path
          d="M60 62 L56 66 L64 66 Z"
          fill="#06b6d4"
          initial={{ scale: 0 }}
          animate={{ scale: 1 }}
          transition={{ duration: 0.4, delay: 3.5 }}
        />

        <defs>
          <linearGradient id="foxGradient" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#6366f1" />
            <stop offset="50%" stopColor="#8b5cf6" />
            <stop offset="100%" stopColor="#06b6d4" />
          </linearGradient>
        </defs>
      </motion.svg>
      
      {/* Enhanced fox glow with pulsing effect */}
      <motion.div 
        className={styles.foxGlow}
        animate={{ 
          scale: [1, 1.2, 1],
          opacity: [0.4, 0.8, 0.4]
        }}
        transition={{ 
          duration: 4,
          repeat: Infinity,
          ease: "easeInOut"
        }}
      />
    </motion.div>
  );

  return (
    <section className={styles.hero} ref={containerRef}>
      <div className={styles.container}>
        <motion.div 
          className={styles.heroContent}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 1 }}
        >
          <div className={styles.heroText}>
            <motion.div 
              className={styles.badge}
              initial={{ opacity: 0, y: -20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.2 }}
            >
              <motion.span 
                className={styles.badgeIcon}
                animate={{ 
                  rotate: [0, 15, -15, 0],
                  scale: [1, 1.2, 1]
                }}
                transition={{ 
                  duration: 3, 
                  repeat: Infinity,
                  repeatDelay: 4
                }}
              >
                ✨
              </motion.span>
              New: AI-powered grammar correction
            </motion.div>
            
            <motion.h1 
              className={styles.title}
              initial={{ opacity: 0, y: 30 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.8, delay: 0.4 }}
            >
              Master Japanese the 
              <motion.span 
                className={styles.gradient}
                initial={{ backgroundPosition: "0% 50%" }}
                animate={{ backgroundPosition: "100% 50%" }}
                transition={{ 
                  duration: 4,
                  repeat: Infinity,
                  repeatType: "reverse"
                }}
              >
                {" "}Smart Way
              </motion.span>
            </motion.h1>
            
            <motion.p 
              className={styles.subtitle}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.6 }}
            >
              From Hiragana to JLPT N1 — Learn with structure, have fun, and see real results. 
              Join thousands transforming their Japanese journey.
            </motion.p>
            
            <motion.div 
              className={styles.heroActions}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.8 }}
            >
              <motion.button 
                className={styles.primaryBtn}
                whileHover={{ 
                  scale: 1.05,
                  boxShadow: "0 25px 50px rgba(99, 102, 241, 0.4)"
                }}
                whileTap={{ scale: 0.95 }}
                transition={{ type: "spring", stiffness: 300 }}
              >
                <span>Start Free Trial</span>
                <motion.svg 
                  className={styles.arrow} 
                  viewBox="0 0 20 20" 
                  fill="currentColor"
                  animate={{ x: [0, 4, 0] }}
                  transition={{ 
                    duration: 2, 
                    repeat: Infinity,
                    repeatDelay: 3
                  }}
                >
                  <path fillRule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clipRule="evenodd" />
                </motion.svg>
              </motion.button>
              <motion.button 
                className={styles.secondaryBtn}
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.98 }}
              >
                Take Placement Test
              </motion.button>
            </motion.div>
            
            <motion.div 
              className={styles.trustIndicators}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.6, delay: 1 }}
            >
              {[
                { icon: "⭐", text: "4.9/5 rating" },
                { icon: "👥", text: "50,000+ learners" },
                { icon: "🎯", text: "JLPT focused" }
              ].map((indicator, index) => (
                <motion.div 
                  key={index}
                  className={styles.indicator}
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ duration: 0.4, delay: 1.2 + index * 0.1 }}
                >
                  <span className={styles.indicatorIcon}>{indicator.icon}</span>
                  <span>{indicator.text}</span>
                </motion.div>
              ))}
            </motion.div>
          </div>
          
          <motion.div 
            className={styles.heroVisual}
            initial={{ opacity: 0, x: 50 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.8, delay: 0.3 }}
          >
            <div className={styles.orbitalSystem}>
              {/* Central Fox */}
              <motion.div 
                className={styles.centralFox}
                animate={{ 
                  y: [0, -12, 0],
                  rotate: [0, 2, -2, 0]
                }}
                transition={{ 
                  duration: 5,
                  repeat: Infinity,
                  ease: "easeInOut"
                }}
              >
                <FoxLogo />
                <motion.div 
                  className={styles.speechBubble}
                  initial={{ opacity: 0, scale: 0 }}
                  animate={{ opacity: 1, scale: 1 }}
                  transition={{ duration: 0.6, delay: 4 }}
                >
                  <motion.span
                    animate={{ 
                      scale: [1, 1.08, 1]
                    }}
                    transition={{ 
                      duration: 3,
                      repeat: Infinity,
                      repeatDelay: 2
                    }}
                  >
                    がんばって！
                  </motion.span>
                </motion.div>
              </motion.div>

              {/* 3D Inner Orbital Ring */}
              <motion.div 
                className={styles.orbitRing} 
                data-ring="inner"
                initial={{ rotateX: 0, rotateY: 0 }}
                animate={{ rotateX: [0, 10, 0], rotateY: [0, 5, 0] }}
                transition={{ duration: 8, repeat: Infinity, ease: "easeInOut" }}
              >
                {orbitalWords.inner.map((word, index) => (
                  <motion.div
                    key={word}
                    className={styles.orbitalWord}
                    style={{ 
                      '--orbit-delay': `${index * 3}s`,
                      '--total-items': orbitalWords.inner.length 
                    } as any}
                    initial={{ opacity: 0, scale: 0 }}
                    animate={{ opacity: 1, scale: 1 }}
                    transition={{ duration: 0.6, delay: 2.5 + index * 0.3 }}
                  >
                    {word}
                  </motion.div>
                ))}
              </motion.div>

              {/* 3D Middle Orbital Ring */}
              <motion.div 
                className={styles.orbitRing} 
                data-ring="middle"
                initial={{ rotateX: 0, rotateY: 0 }}
                animate={{ rotateX: [0, -8, 0], rotateY: [0, -10, 0] }}
                transition={{ duration: 12, repeat: Infinity, ease: "easeInOut" }}
              >
                {orbitalWords.middle.map((word, index) => (
                  <motion.div
                    key={word}
                    className={styles.orbitalWord}
                    style={{ 
                      '--orbit-delay': `${index * 2}s`,
                      '--total-items': orbitalWords.middle.length 
                    } as any}
                    initial={{ opacity: 0, scale: 0 }}
                    animate={{ opacity: 1, scale: 1 }}
                    transition={{ duration: 0.6, delay: 3 + index * 0.2 }}
                  >
                    {word}
                  </motion.div>
                ))}
              </motion.div>

              {/* 3D Outer Orbital Ring */}
              <motion.div 
                className={styles.orbitRing} 
                data-ring="outer"
                initial={{ rotateX: 0, rotateY: 0 }}
                animate={{ rotateX: [0, 12, 0], rotateY: [0, 8, 0] }}
                transition={{ duration: 15, repeat: Infinity, ease: "easeInOut" }}
              >
                {orbitalWords.outer.map((word, index) => (
                  <motion.div
                    key={word}
                    className={styles.orbitalWord}
                    style={{ 
                      '--orbit-delay': `${index * 1.5}s`,
                      '--total-items': orbitalWords.outer.length 
                    } as any}
                    initial={{ opacity: 0, scale: 0 }}
                    animate={{ opacity: 1, scale: 1 }}
                    transition={{ duration: 0.6, delay: 3.5 + index * 0.15 }}
                  >
                    {word}
                  </motion.div>
                ))}
              </motion.div>
            </div>
          </motion.div>
        </motion.div>
      </div>

      {/* Water Bubbles with Burst Animation */}
      <AnimatePresence>
        {bubbles.map((bubble) => (
          <motion.div
            key={bubble.id}
            className={styles.waterBubble}
            style={{
              left: bubble.x,
              top: bubble.y,
              width: bubble.size,
              height: bubble.size,
              background: `radial-gradient(circle at 30% 30%, rgba(255,255,255,0.8), ${bubble.color})`,
              opacity: bubble.opacity
            }}
            initial={{ scale: 0, opacity: 0 }}
            animate={{ 
              scale: burstAnimations[bubble.id] ? [1, 1.5, 0] : 1,
              opacity: burstAnimations[bubble.id] ? [bubble.opacity, 1, 0] : bubble.opacity 
            }}
            exit={{ scale: 0, opacity: 0 }}
            transition={{ 
              scale: burstAnimations[bubble.id] ? { duration: 0.3 } : { duration: 0.5 },
              opacity: burstAnimations[bubble.id] ? { duration: 0.3 } : { duration: 0.5 }
            }}
            onClick={() => handleBubbleBurst(bubble)}
            whileHover={{ scale: 1.1 }}
            whileTap={{ scale: 0.9 }}
          />
        ))}
      </AnimatePresence>

      {/* Enhanced Firecracker Particles */}
      <AnimatePresence>
        {particles.map((particle) => (
          <motion.div
            key={particle.id}
            className={styles.burstParticle}
            style={{
              left: particle.x,
              top: particle.y,
              width: particle.size,
              height: particle.size,
              background: `radial-gradient(circle, ${particle.color}, transparent)`,
              opacity: particle.life
            }}
            initial={{ scale: 1 }}
            exit={{ scale: 0 }}
            animate={{ rotate: 360 }}
            transition={{ duration: 2, ease: "linear" }}
          />
        ))}
      </AnimatePresence>
    </section>
  );
};

export default HeroSection;
