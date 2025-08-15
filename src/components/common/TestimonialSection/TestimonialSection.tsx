import React from 'react';
import styles from './TestimonialSection.module.css';

const TestimonialSection: React.FC = () => {
  const testimonials = [
    {
      name: 'Sarah Chen',
      role: 'Software Engineer',
      text: 'FluentFox made kanji learning actually enjoyable. I went from struggling with basic characters to reading manga in just 8 months!',
      avatar: '👩‍💻'
    },
    {
      name: 'Michael Rodriguez',
      role: 'Student',
      text: 'The spaced repetition system is genius. I passed JLPT N3 after using FluentFox for one year. Highly recommended!',
      avatar: '👨‍🎓'
    },
    {
      name: 'Emma Thompson',
      role: 'Business Analyst',
      text: 'Living in Japan became so much easier after learning kanji with FluentFox. I can now read signs, menus, and news effortlessly.',
      avatar: '👩‍💼'
    },
  ];

  return (
    <section className={styles.testimonials}>
      <div className={styles.container}>
        <div className={styles.header}>
          <h2 className={styles.title}>Success Stories</h2>
          <p className={styles.subtitle}>
            Join thousands who've transformed their Japanese learning
          </p>
        </div>
        <div className={styles.testimonialGrid}>
          {testimonials.map((testimonial, index) => (
            <div key={index} className={styles.testimonialCard}>
              <div className={styles.testimonialContent}>
                <p className={styles.testimonialText}>"{testimonial.text}"</p>
              </div>
              <div className={styles.testimonialAuthor}>
                <span className={styles.avatar}>{testimonial.avatar}</span>
                <div className={styles.authorInfo}>
                  <div className={styles.authorName}>{testimonial.name}</div>
                  <div className={styles.authorRole}>{testimonial.role}</div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default TestimonialSection;
