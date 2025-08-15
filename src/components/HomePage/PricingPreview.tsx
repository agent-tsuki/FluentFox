// components/PricingPreview.tsx
import React, { useState } from 'react';
import styles from './PricingPreview.module.css';

const plans = [
  {
    name: 'Free',
    price: 0,
    period: 'forever',
    features: [
      'Basic Hiragana & Katakana lessons',
      '50 Kanji characters',
      'Community forum access',
      'Mobile app'
    ],
    cta: 'Get Started',
    popular: false
  },
  {
    name: 'Pro',
    price: 12,
    period: 'month',
    yearlyPrice: 96,
    features: [
      'All Free features',
      'Complete JLPT N5-N1 content',
      'AI-powered corrections',
      'Progress analytics',
      'Offline mode',
      'Priority support'
    ],
    cta: 'Start Free Trial',
    popular: true
  },
  {
    name: 'Premium',
    price: 24,
    period: 'month',
    yearlyPrice: 192,
    features: [
      'All Pro features',
      'Personal tutor sessions',
      'Custom study plans',
      'Advanced analytics',
      'Certificate of completion',
      '1-on-1 speaking practice'
    ],
    cta: 'Start Free Trial',
    popular: false
  }
];

const PricingPreview: React.FC = () => {
  const [isYearly, setIsYearly] = useState(false);

  return (
    <section className={styles.pricing}>
      <div className={styles.container}>
        <div className={styles.sectionHeader}>
          <h2 className={styles.title}>
            Choose Your <span className={styles.gradient}>Learning Plan</span>
          </h2>
          <p className={styles.subtitle}>
            Start free, upgrade anytime. No commitments, just results.
          </p>
          
          <div className={styles.billingToggle}>
            <span className={!isYearly ? styles.active : ''}>Monthly</span>
            <button 
              className={styles.toggle}
              onClick={() => setIsYearly(!isYearly)}
            >
              <div className={`${styles.toggleSlider} ${isYearly ? styles.yearly : ''}`}></div>
            </button>
            <span className={isYearly ? styles.active : ''}>
              Yearly
              <span className={styles.discount}>Save 33%</span>
            </span>
          </div>
        </div>
        
        <div className={styles.plansGrid}>
          {plans.map((plan, index) => (
            <div 
              key={index}
              className={`${styles.planCard} ${plan.popular ? styles.popular : ''}`}
            >
              {plan.popular && (
                <div className={styles.popularBadge}>
                  <span>Most Popular</span>
                </div>
              )}
              
              <div className={styles.planHeader}>
                <h3 className={styles.planName}>{plan.name}</h3>
                <div className={styles.planPrice}>
                  <span className={styles.currency}>$</span>
                  <span className={styles.amount}>
                    {plan.price === 0 ? '0' : isYearly && plan.yearlyPrice ? Math.floor(plan.yearlyPrice / 12) : plan.price}
                  </span>
                  <span className={styles.period}>/{plan.period}</span>
                </div>
                {isYearly && plan.yearlyPrice && (
                  <div className={styles.yearlyPrice}>
                    Billed ${plan.yearlyPrice} annually
                  </div>
                )}
              </div>
              
              <ul className={styles.featuresList}>
                {plan.features.map((feature, featureIndex) => (
                  <li key={featureIndex} className={styles.feature}>
                    <svg className={styles.checkIcon} viewBox="0 0 20 20" fill="currentColor">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                    {feature}
                  </li>
                ))}
              </ul>
              
              <button className={`${styles.planCTA} ${plan.popular ? styles.primaryCTA : styles.secondaryCTA}`}>
                {plan.cta}
              </button>
            </div>
          ))}
        </div>
        
        <div className={styles.guarantee}>
          <span className={styles.guaranteeIcon}>🛡️</span>
          <span>30-day money-back guarantee • No questions asked</span>
        </div>
      </div>
    </section>
  );
};

export default PricingPreview;
