import React, { useState, useEffect, useRef } from 'react';
import styles from './Navbar.module.css';

type LinkItem = { label: string; href: string };

const navigationItems = [
  {
    label: 'Alphabet',
    items: [
      { label: 'Hiragana & Katakana', href: '/alphabet' },
      { label: 'Stroke Order', href: '/alphabet/stroke-order' },
      { label: 'Audio Practice', href: '/alphabet/audio' },
      { label: 'Typing Practice', href: '/alphabet/typing' },
    ],
  },
  {
    label: 'Kanji',
    items: [
      { label: 'JLPT Levels', href: '/kanji/jlpt' },
      { label: 'By Radicals', href: '/kanji/radicals' },
      { label: 'Stroke Animator', href: '/kanji/stroke-animator' },
      { label: 'Writing Practice', href: '/kanji/writing' },
    ],
  },
  {
    label: 'Grammar',
    items: [
      { label: 'JLPT Grammar', href: '/grammar/jlpt' },
      { label: 'Essential Patterns', href: '/grammar/patterns' },
      { label: 'Particles Guide', href: '/grammar/particles' },
      { label: 'Practice Quizzes', href: '/grammar/practice' },
    ],
  },
  {
    label: 'JLPT',
    items: [
      { label: 'Mock Tests', href: '/jlpt/mock-tests' },
      { label: 'Study Plans', href: '/jlpt/study-plans' },
      { label: 'Progress Tracker', href: '/jlpt/progress' },
      { label: 'Exam Tips', href: '/jlpt/tips' },
    ],
  },
];

const Navbar: React.FC = () => {
  const [isScrolled, setIsScrolled] = useState(false);
  const [openDropdown, setOpenDropdown] = useState<string | null>(null);
  const [mobileOpen, setMobileOpen] = useState(false);
  const navRef = useRef<HTMLDivElement | null>(null);

  // Handle scroll effect
  useEffect(() => {
    const onScroll = () => {
      setIsScrolled(window.scrollY > 20);
    };

    window.addEventListener('scroll', onScroll);
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  // Handle click outside to close dropdowns
  useEffect(() => {
    const onClickOutside = (e: MouseEvent) => {
      if (navRef.current && !navRef.current.contains(e.target as Node)) {
        setOpenDropdown(null);
      }
    };

    document.addEventListener('mousedown', onClickOutside);
    return () => document.removeEventListener('mousedown', onClickOutside);
  }, []);

  const toggleDropdown = (label: string) => {
    setOpenDropdown(openDropdown === label ? null : label);
  };

  const closeAllDropdowns = () => {
    setOpenDropdown(null);
    setMobileOpen(false);
  };

  return (
    <nav
      className={`${styles.navbar} ${isScrolled ? styles.scrolled : ''}`}
      ref={navRef}
    >
      <div className={styles.container}>
        {/* Logo */}
        <a href="/" className={styles.logo}>
        <div className={styles.logoIcon}>
            🦊
        </div>
        <span className={styles.logoText}>FluentFox</span>
        </a>

        {/* Desktop Navigation */}
        <div className={styles.desktopNav}>
          {navigationItems.map((item) => (
            <div
              key={item.label}
              className={styles.navItem}
              onMouseEnter={() => setOpenDropdown(item.label)}
              onMouseLeave={() => setOpenDropdown(null)}
            >
              <button
                className={styles.navButton}
                onClick={() => toggleDropdown(item.label)}
              >
                {item.label}
                <svg className={styles.chevron} viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clipRule="evenodd" />
                </svg>
              </button>

              <div className={`${styles.dropdown} ${openDropdown === item.label ? styles.open : ''}`}>
                <div className={styles.dropdownContent}>
                  {item.items.map((subItem) => (
                    <a
                      key={subItem.label}
                      href={subItem.href}
                      className={styles.dropdownLink}
                      onClick={closeAllDropdowns}
                    >
                      {subItem.label}
                    </a>
                  ))}
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Action Buttons */}
        <div className={styles.actions}>
          <button className={styles.loginBtn}>Login</button>
          <button className={styles.signupBtn}>Signup</button>
        </div>

        {/* Mobile Menu Button */}
        <button
          className={`${styles.mobileMenuBtn} ${mobileOpen ? styles.open : ''}`}
          onClick={() => setMobileOpen(!mobileOpen)}
          aria-label="Toggle menu"
        >
          <span></span>
          <span></span>
          <span></span>
        </button>
      </div>

      {/* Mobile Menu */}
      <div className={`${styles.mobileMenu} ${mobileOpen ? styles.open : ''}`}>
        <div className={styles.mobileContent}>
          {navigationItems.map((item) => (
            <details key={item.label} className={styles.mobileSection}>
              <summary className={styles.mobileSectionTitle}>{item.label}</summary>
              <div className={styles.mobileLinks}>
                {item.items.map((subItem) => (
                  <a
                    key={subItem.label}
                    href={subItem.href}
                    className={styles.mobileLink}
                    onClick={closeAllDropdowns}
                  >
                    {subItem.label}
                  </a>
                ))}
              </div>
            </details>
          ))}
          <div className={styles.mobileActions}>
            <button className={styles.mobileLoginBtn} onClick={closeAllDropdowns}>
              Login
            </button>
            <button className={styles.mobileSignupBtn} onClick={closeAllDropdowns}>
              Signup
            </button>
          </div>
        </div>
      </div>
    </nav>
  );
};

export default Navbar;
