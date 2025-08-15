import React from 'react';
import styles from './Footer.module.css';

const Footer: React.FC = () => {
  return (
    <footer className={styles.footer}>
      <div className={styles.container}>
        <div className={styles.footerContent}>
          {/* Brand Section */}
          <div className={styles.brandSection}>
            <div className={styles.brand}>
              <span className={styles.brandIcon}>🦊</span>
              <span className={styles.brandText}>FluentFox</span>
            </div>
            <p className={styles.brandDescription}>
              Master Japanese through intelligent learning. Join thousands who've transformed their language journey with our comprehensive platform.
            </p>
            <div className={styles.socialLinks}>
              <a href="#" className={styles.socialLink} aria-label="Follow us on Twitter">
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path d="M23.953 4.57a10 10 0 01-2.825.775 4.958 4.958 0 002.163-2.723c-.951.555-2.005.959-3.127 1.184a4.92 4.92 0 00-8.384 4.482C7.69 8.095 4.067 6.13 1.64 3.162a4.822 4.822 0 00-.666 2.475c0 1.71.87 3.213 2.188 4.096a4.904 4.904 0 01-2.228-.616v.06a4.923 4.923 0 003.946 4.827 4.996 4.996 0 01-2.212.085 4.936 4.936 0 004.604 3.417 9.867 9.867 0 01-6.102 2.105c-.39 0-.779-.023-1.17-.067a13.995 13.995 0 007.557 2.209c9.053 0 13.998-7.496 13.998-13.985 0-.21 0-.42-.015-.63A9.935 9.935 0 0024 4.59z"/>
                </svg>
              </a>
              <a href="#" className={styles.socialLink} aria-label="Join our Discord">
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path d="M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 00.0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 00-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 01-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 01.0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 01.0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 01-.0066.1276 12.2986 12.2986 0 01-1.873.8914.0766.0766 0 00-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 00.0842.0286c1.961-.6067 3.9495-1.5219 6.0023-3.0294a.077.077 0 00.0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 00-.0312-.0286zM8.02 15.3312c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9555-2.4189 2.157-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419-.0188 1.3332-.9555 2.4189-2.1569 2.4189zm7.9748 0c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9554-2.4189 2.1569-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.9554 2.4189-2.1568 2.4189Z"/>
                </svg>
              </a>
              <a href="#" className={styles.socialLink} aria-label="Join our Reddit community">
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 0A12 12 0 0 0 0 12a12 12 0 0 0 12 12 12 12 0 0 0 12-12A12 12 0 0 0 12 0zm5.01 4.744c.688 0 1.25.561 1.25 1.249a1.25 1.25 0 0 1-2.498.056l-2.597-.547-.8 3.747c1.824.07 3.48.632 4.674 1.488.308-.309.73-.491 1.207-.491.968 0 1.754.786 1.754 1.754 0 .716-.435 1.333-1.01 1.614a3.111 3.111 0 0 1 .042.52c0 2.694-3.13 4.87-7.004 4.87-3.874 0-7.004-2.176-7.004-4.87 0-.183.015-.366.043-.534A1.748 1.748 0 0 1 4.028 12c0-.968.786-1.754 1.754-1.754.463 0 .898.196 1.207.49 1.207-.883 2.878-1.43 4.744-1.487l.885-4.182a.342.342 0 0 1 .14-.197.35.35 0 0 1 .238-.042l2.906.617a1.214 1.214 0 0 1 1.108-.701zM9.25 12C8.561 12 8 12.562 8 13.25c0 .687.561 1.248 1.25 1.248.687 0 1.248-.561 1.248-1.249 0-.688-.561-1.249-1.249-1.249zm5.5 0c-.687 0-1.248.561-1.248 1.25 0 .687.561 1.248 1.249 1.248.688 0 1.249-.561 1.249-1.249 0-.687-.562-1.249-1.25-1.249zm-5.466 3.99a.327.327 0 0 0-.231.094.33.33 0 0 0 0 .463c.842.842 2.484.913 2.961.913.477 0 2.105-.056 2.961-.913a.361.361 0 0 0 .029-.463.33.33 0 0 0-.464 0c-.547.533-1.684.73-2.512.73-.828 0-1.979-.196-2.512-.73a.326.326 0 0 0-.232-.095z"/>
                </svg>
              </a>
              <a href="#" className={styles.socialLink} aria-label="Subscribe to our YouTube channel">
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path d="M23.498 6.186a3.016 3.016 0 0 0-2.122-2.136C19.505 3.545 12 3.545 12 3.545s-7.505 0-9.377.505A3.017 3.017 0 0 0 .502 6.186C0 8.07 0 12 0 12s0 3.93.502 5.814a3.016 3.016 0 0 0 2.122 2.136c1.871.505 9.376.505 9.376.505s7.505 0 9.377-.505a3.015 3.015 0 0 0 2.122-2.136C24 15.93 24 12 24 12s0-3.93-.502-5.814zM9.545 15.568V8.432L15.818 12l-6.273 3.568z"/>
                </svg>
              </a>
            </div>
          </div>
          
          {/* Links Sections */}
          <div className={styles.linksGrid}>
            <div className={styles.linkSection}>
              <h4 className={styles.sectionTitle}>Learning</h4>
              <ul className={styles.linkList}>
                <li><a href="/alphabet" className={styles.footerLink}>Hiragana & Katakana</a></li>
                <li><a href="/kanji" className={styles.footerLink}>Kanji Builder</a></li>
                <li><a href="/grammar" className={styles.footerLink}>Grammar Guide</a></li>
                <li><a href="/vocabulary" className={styles.footerLink}>Vocabulary</a></li>
                <li><a href="/jlpt" className={styles.footerLink}>JLPT Prep</a></li>
                <li><a href="/practice" className={styles.footerLink}>Practice Tests</a></li>
              </ul>
            </div>
            
            <div className={styles.linkSection}>
              <h4 className={styles.sectionTitle}>Resources</h4>
              <ul className={styles.linkList}>
                <li><a href="/blog" className={styles.footerLink}>Learning Blog</a></li>
                <li><a href="/guides" className={styles.footerLink}>Study Guides</a></li>
                <li><a href="/tools" className={styles.footerLink}>Learning Tools</a></li>
                <li><a href="/help" className={styles.footerLink}>Help Center</a></li>
                <li><a href="/api" className={styles.footerLink}>Developer API</a></li>
                <li><a href="/community" className={styles.footerLink}>Community</a></li>
              </ul>
            </div>
            
            <div className={styles.linkSection}>
              <h4 className={styles.sectionTitle}>Company</h4>
              <ul className={styles.linkList}>
                <li><a href="/about" className={styles.footerLink}>About Us</a></li>
                <li><a href="/careers" className={styles.footerLink}>Careers</a></li>
                <li><a href="/press" className={styles.footerLink}>Press Kit</a></li>
                <li><a href="/contact" className={styles.footerLink}>Contact</a></li>
                <li><a href="/partners" className={styles.footerLink}>Partners</a></li>
                <li><a href="/affiliate" className={styles.footerLink}>Affiliate Program</a></li>
              </ul>
            </div>
            
            <div className={styles.linkSection}>
              <h4 className={styles.sectionTitle}>Legal</h4>
              <ul className={styles.linkList}>
                <li><a href="/privacy" className={styles.footerLink}>Privacy Policy</a></li>
                <li><a href="/terms" className={styles.footerLink}>Terms of Service</a></li>
                <li><a href="/cookies" className={styles.footerLink}>Cookie Policy</a></li>
                <li><a href="/dmca" className={styles.footerLink}>DMCA</a></li>
              </ul>
            </div>
          </div>
        </div>

        {/* Newsletter Section */}
        <div className={styles.newsletterSection}>
          <div className={styles.newsletterContent}>
            <h3 className={styles.newsletterTitle}>Stay Updated</h3>
            <p className={styles.newsletterDescription}>
              Get the latest Japanese learning tips, new features, and exclusive content delivered to your inbox.
            </p>
          </div>
          <div className={styles.newsletterForm}>
            <div className={styles.inputGroup}>
              <input 
                type="email" 
                placeholder="Enter your email address" 
                className={styles.emailInput}
                aria-label="Email address"
              />
              <button type="submit" className={styles.subscribeBtn}>
                Subscribe
                <svg className={styles.arrowIcon} viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clipRule="evenodd" />
                </svg>
              </button>
            </div>
          </div>
        </div>
        
        {/* Footer Bottom */}
        <div className={styles.footerBottom}>
          <div className={styles.copyright}>
            <p>© 2024 FluentFox. All rights reserved. Made with ❤️ for Japanese learners worldwide.</p>
          </div>
          <div className={styles.badges}>
            <div className={styles.badge}>
              <span className={styles.badgeIcon}>⭐</span>
              <span>4.9/5 User Rating</span>
            </div>
            <div className={styles.badge}>
              <span className={styles.badgeIcon}>🌏</span>
              <span>50K+ Active Learners</span>
            </div>
          </div>
        </div>
      </div>
    </footer>
  );
};

export default Footer;
