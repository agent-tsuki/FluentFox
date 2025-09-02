// Home.tsx
import React from 'react';
import HeroSection from '../../components/HomePage/HeroSection';
import FeaturesSection from '../../components/HomePage/FeaturesSection';
import LearningPathSection from '../../components/HomePage/LearningPathSection';
import SocialProofSection from '../../components/HomePage/SocialProofSection';
import HowItWorksSection from '../../components/HomePage/HowItWorksSection';
import PricingPreview from '../../components/HomePage/PricingPreview';
import styles from '../../styles/pages/Home.module.css';

const HomePage: React.FC = () => {
  return (
    <div className={styles.homePage}>
      <HeroSection />
      <FeaturesSection />
      <LearningPathSection />
      <SocialProofSection />
      <HowItWorksSection />
      <PricingPreview />
    </div>
  );
};

export default HomePage;
