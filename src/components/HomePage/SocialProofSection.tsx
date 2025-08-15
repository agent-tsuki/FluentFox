// components/SocialProofSection.tsx
import React from 'react';
import styles from './SocialProofSection.module.css';

const stats = [
  { number: '50,000+', label: 'Active Learners', icon: '👥' },
  { number: '4.9/5', label: 'User Rating', icon: '⭐' },
  { number: '95%', label: 'JLPT Pass Rate', icon: '🎯' },
  { number: '1M+', label: 'Lessons Completed', icon: '📚' }
];

const testimonials = [
  {
    name: 'Sarah Chen',
    role: 'Software Engineer',
    avatar: '👩‍💻',
    text: 'FluentFox helped me pass JLPT N3 in just 6 months. The structured approach and AI feedback made all the difference!',
    rating: 5
  },
  {
    name: 'Mike Johnson',
    role: 'Anime Enthusiast',
    avatar: '🧑‍🎨',
    text: 'Finally understanding anime without subtitles! The gamified lessons keep me motivated every day.',
    rating: 5
  },
  {
    name: 'Yuki Tanaka',
    role: 'Japanese Tutor',
    avatar: '👨‍🏫',
    text: 'I recommend FluentFox to all my students. The grammar explanations are clear and the practice exercises are excellent.',
    rating: 5
  }
];

const SocialProofSection: React.FC = () => {
  return (
    <section className={styles.socialProof}>
      <div className={styles.container}>
        <div className={styles.statsSection}>
          <div className={styles.statsGrid}>
            {stats.map((stat, index) => (
              <div key={index} className={styles.statCard}>
                <div className={styles.statIcon}>{stat.icon}</div>
                <div className={styles.statNumber}>{stat.number}</div>
                <div className={styles.statLabel}>{stat.label}</div>
              </div>
            ))}
          </div>
        </div>
        
        <div className={styles.testimonialsSection}>
          <h2 className={styles.title}>
            What Our <span className={styles.gradient}>Learners Say</span>
          </h2>
          
          <div className={styles.testimonialsGrid}>
            {testimonials.map((testimonial, index) => (
              <div key={index} className={styles.testimonialCard}>
                <div className={styles.testimonialRating}>
                  {[...Array(testimonial.rating)].map((_, i) => (
                    <span key={i} className={styles.star}>⭐</span>
                  ))}
                </div>
                <p className={styles.testimonialText}>"{testimonial.text}"</p>
                <div className={styles.testimonialAuthor}>
                  <div className={styles.authorAvatar}>{testimonial.avatar}</div>
                  <div className={styles.authorInfo}>
                    <div className={styles.authorName}>{testimonial.name}</div>
                    <div className={styles.authorRole}>{testimonial.role}</div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
};

export default SocialProofSection;
