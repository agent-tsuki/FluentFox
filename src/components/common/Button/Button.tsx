import React from 'react';
import styles from '../../../styles/components/Button.module.css';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'destructive' | 'kanji' | 'hiragana' | 'katakana' | 'grammar' | 'jlpt';
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  isLoading?: boolean;
  isDisabled?: boolean;
  fullWidth?: boolean;
  iconOnly?: boolean;
  fab?: boolean;
  glass?: boolean;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
  children?: React.ReactNode;
  as?: 'button' | 'a';
  href?: string;
}

const Button: React.FC<ButtonProps> = ({
  variant = 'primary',
  size = 'md',
  isLoading = false,
  isDisabled = false,
  fullWidth = false,
  iconOnly = false,
  fab = false,
  glass = false,
  leftIcon,
  rightIcon,
  children,
  className,
  as = 'button',
  href,
  ...props
}) => {
  const classNames = [
    styles.button,
    styles[variant],
    styles[size],
    isLoading && styles.loading,
    fullWidth && styles.fullWidth,
    iconOnly && styles.iconOnly,
    fab && styles.fab,
    glass && styles.glass,
    className
  ].filter(Boolean).join(' ');

  const content = (
    <>
      {leftIcon && <span className={styles.leftIcon}>{leftIcon}</span>}
      {!isLoading && children}
      {rightIcon && <span className={styles.rightIcon}>{rightIcon}</span>}
    </>
  );

  if (as === 'a') {
    return (
      <a
        href={href}
        className={classNames}
        aria-disabled={isDisabled}
        {...(props as React.AnchorHTMLAttributes<HTMLAnchorElement>)}
      >
        {content}
      </a>
    );
  }

  return (
    <button
      className={classNames}
      disabled={isDisabled || isLoading}
      {...props}
    >
      {content}
    </button>
  );
};

export default Button;